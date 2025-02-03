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
	"strings"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
)

type FileCache struct {
	memoryCache *memoryCache
	filename    string
	needsSave   bool
}

// NewFileCacheWithContent creates a FileCache with existing content, e.g. from an embedded file.
//
// Optionally, set a filename and call FileCache.Save to save the file.
func NewFileCacheWithContent(content []byte, filename string) (*FileCache, error) {
	c := &FileCache{filename: filename, memoryCache: &memoryCache{}}
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
	c := &FileCache{filename: filename, memoryCache: &memoryCache{}}

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
	f, err := os.Create(c.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	c.memoryCache.mutex.RLock()
	defer c.memoryCache.mutex.RUnlock()
	if c.memoryCache.cache == nil {
		c.memoryCache.cache = make(map[string]*cel.Ast)
	}
	for k, v := range c.memoryCache.cache {
		m, err := marshalAST(v)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(f, "%s\t%s\n", k, m); err != nil {
			return err
		}
	}
	return nil
}

func cleanExpr(expr string) string {
	return strings.ReplaceAll(expr, "\t", " ")
}

func (c *FileCache) load(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	unmarshalled := make(map[string]*cel.Ast)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return errors.New("malformed file: line does not contain two parts")
		}
		ast, err := unmarshalAST(parts[1])
		if err != nil {
			return fmt.Errorf("could not unmarshal cached data at line %d: %w", lineNumber, err)
		}
		unmarshalled[parts[0]] = ast
	}
	c.memoryCache.mutex.Lock()
	c.memoryCache.cache = unmarshalled
	c.memoryCache.mutex.Unlock()
	return nil
}

func (c *FileCache) Get(expr string) (*cel.Ast, bool) {
	return c.memoryCache.Get(cleanExpr(expr))
}

func (c *FileCache) Set(expr string, ast *cel.Ast) error {
	c.needsSave = true
	return c.memoryCache.Set(cleanExpr(expr), ast)
}

func marshalAST(ast *cel.Ast) (string, error) {
	parsedExpr, err := cel.AstToParsedExpr(ast)
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
	var m = &exprpb.ParsedExpr{}
	if err := proto.Unmarshal(b, m); err != nil {
		return nil, err
	}

	return cel.ParsedExprToAst(m), nil
}
