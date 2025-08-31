package rules

// RulesetSpec defines a ruleset.
type RulesetSpec interface {
	GetName() string
	GetRules() []RuleSpec
}

// RuleSpec defines a rule.
type RuleSpec interface {
	GetName() string
	GetCondition() string
	GetResults() []string
}

// Ruleset is the default implementation of a ruleset (see RulesetSpec).
type Ruleset struct {
	Name  string     `yaml:"name,omitempty"`
	Rules []RuleSpec `yaml:"rules"`
}

func (r *Ruleset) GetName() string      { return r.Name }
func (r *Ruleset) GetRules() []RuleSpec { return r.Rules }

// Rule is the default implementation of a rule (see RuleSpec).
type Rule struct {
	Name string `yaml:"name,omitempty"`

	When  string           `yaml:"when"`
	Then  YAMLListOrString `yaml:"then"`
	Maybe YAMLListOrString `yaml:"maybe"`

	With map[string]string `yaml:"with"`

	Group     string           `yaml:"group"`
	GroupList YAMLListOrString `yaml:"groups"`

	Ignore YAMLListOrString `yaml:"ignore"`

	ReadFiles []string `yaml:"read_files"`
}

func (r *Rule) GetMetadata() map[string]string {
	return r.With
}

func (r *Rule) GetName() string {
	return r.Name
}

func (r *Rule) GetCondition() string {
	return r.When
}

func (r *Rule) GetResults() []string {
	return r.Then
}

func (r *Rule) GetMaybeResults() []string {
	return r.Maybe
}

func (r *Rule) GetGroups() []string {
	if r.Group != "" {
		return []string{r.Group}
	}
	return r.GroupList
}

func (r *Rule) GetIgnores() []string {
	return r.Ignore
}

func (r *Rule) GetReadFiles() []string {
	return r.ReadFiles
}

// WithMaybeResults adds to a RuleSpec the possibility of a rule having uncertain results.
type WithMaybeResults interface {
	GetMaybeResults() []string
}

// WithGroups adds to a RuleSpec the feature of a rule having groups.
type WithGroups interface {
	GetGroups() []string
}

// WithMetadata adds to a RuleSpec the feature of a rule having metadata.
type WithMetadata interface {
	GetMetadata() map[string]string
}

type WithReadFiles interface {
	GetReadFiles() []string
}
