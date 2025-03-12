package eval

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
)

type FileCache struct {
	exprCache sync.Map
	filename  string
	needsSave bool
}

// NewFileCacheWithContent creates a FileCache with existing content, e.g. from an embedded file.
//
// Optionally, set a filename and call FileCache.Save to save the file.
func NewFileCacheWithContent(content []byte, filename string) (*FileCache, error) {
	c := &FileCache{filename: filename}
	if err := c.load(bytes.NewReader(content)); err != nil {
		return nil, err
	}
	return c, nil
}

// NewFileCache creates a Cache that can persist to a file.
//
// This function will read the file. The caller will need to run the
// FileCache.Save method to save the file.
func NewFileCache(filename string) (*FileCache, error) {
	c := &FileCache{filename: filename}

	f, err := os.Open(c.filename)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	} else {
		defer f.Close()
		if err := c.load(f); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *FileCache) Save() error {
	if !c.needsSave {
		return nil
	}
	if c.filename == "" {
		return errors.New("no cache filename specified")
	}
	f, err := os.OpenFile(c.filename, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read the file in order to skip re-marshalling the AST for previously cached expressions.
	// This is for consistency (reducing unnecessary Git diffs) rather than performance.
	// TODO why does the marshalled representation change each time? - perhaps it's due to pointers in the environment
	var (
		marshalled = make(map[string]string)
		scanner    = bufio.NewScanner(f)
		lineNumber = 0
	)
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return fmt.Errorf("malformed line %d (not tab-separated) in file: %s", lineNumber, c.filename)
		}
		if _, ok := c.Get(parts[0]); ok {
			marshalled[parts[0]] = parts[1]
		}
	}
	c.exprCache.Range(func(k, v any) bool {
		// Only re-marshal for existing expressions.
		if _, ok := marshalled[k.(string)]; !ok { //nolint:errcheck // the cached key type is known
			s, _err := marshalAST(v.(*cel.Ast)) //nolint:errcheck // the cached value type is known
			if _err != nil {
				err = _err
				return false
			}
			marshalled[k.(string)] = s //nolint:errcheck // the cached key type is known
		}
		return true
	})
	lines := make([]string, len(marshalled))
	i := 0
	for k, v := range marshalled {
		lines[i] = k + "\t" + v
		i++
	}
	// Sort the result as the cache may be saved to Git.
	slices.Sort(lines)
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	_, err = fmt.Fprint(f, strings.Join(lines, "\n"))
	return err
}

var cleanExpr = strings.NewReplacer("\n", " ", "\t", " ").Replace

func (c *FileCache) load(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return fmt.Errorf("malformed line %d (not tab-separated)", lineNumber)
		}
		ast, err := unmarshalAST(parts[1])
		if err != nil {
			return fmt.Errorf("could not unmarshal cached data at line %d: %w", lineNumber, err)
		}
		c.exprCache.Store(parts[0], ast)
	}
	return nil
}

func (c *FileCache) Get(expr string) (*cel.Ast, bool) {
	if a, ok := c.exprCache.Load(cleanExpr(expr)); ok {
		return a.(*cel.Ast), true //nolint:errcheck // the cache type is known
	}
	return nil, false
}

func (c *FileCache) Set(expr string, ast *cel.Ast) error {
	c.needsSave = true
	c.exprCache.Store(cleanExpr(expr), ast)
	return nil
}

func marshalAST(ast *cel.Ast) (string, error) {
	parsedExpr, err := cel.AstToCheckedExpr(ast)
	if err != nil {
		return "", err
	}
	b, err := proto.Marshal(parsedExpr)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func unmarshalAST(str string) (*cel.Ast, error) {
	b, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}
	var m = &exprpb.CheckedExpr{}
	if err := proto.Unmarshal(b, m); err != nil {
		return nil, err
	}

	return cel.CheckedExprToAst(m), nil
}
