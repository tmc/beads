package gitnotes

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	configRefPrefix = "refs/notes/beads/config"
	issueRefPrefix  = "refs/notes/beads/issues/"
	anchorPayload   = "bdzero-anchor\n"
	version         = 1
)

var (
	ErrNotGitRepo     = errors.New("not a git repository")
	ErrNotFound       = errors.New("bead not found")
	ErrNotInitialized = errors.New("bdzero is not initialized")
	ErrAlreadyInited  = errors.New("bdzero is already initialized")
)

type Store struct {
	repo   string
	anchor string
}

type Config struct {
	Version   int       `json:"version"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	Author    string    `json:"author,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type Bead struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description,omitempty"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	Priority     int        `json:"priority"`
	Labels       []string   `json:"labels,omitempty"`
	Dependencies []string   `json:"dependencies,omitempty"`
	CloseReason  string     `json:"close_reason,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

type CreateInput struct {
	Title        string
	Description  string
	Type         string
	Priority     int
	Labels       []string
	Dependencies []string
}

type UpdateInput struct {
	Title        *string
	Description  *string
	Type         *string
	Priority     *int
	Labels       []string
	AddLabels    []string
	RemoveLabels []string
}

type Filter struct {
	Status string
	Type   string
	Label  string
	Query  string
	All    bool
}

func Open(repo string) (*Store, error) {
	root, err := resolveRepo(repo)
	if err != nil {
		return nil, err
	}
	anchor, err := ensureAnchor(root)
	if err != nil {
		return nil, err
	}
	return &Store{repo: root, anchor: anchor}, nil
}

func (s *Store) Repo() string {
	return s.repo
}

func (s *Store) Init(prefix string, force bool) (*Config, error) {
	if prefix == "" {
		prefix = filepath.Base(s.repo)
	}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		prefix = "bead"
	}
	if cfg, err := s.Config(); err == nil && !force {
		return nil, fmt.Errorf("%w (prefix %q)", ErrAlreadyInited, cfg.Prefix)
	} else if err != nil && !errors.Is(err, ErrNotInitialized) {
		return nil, err
	}

	cfg := &Config{
		Version:   version,
		Prefix:    prefix,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.writeJSON(configRefPrefix, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *Store) Config() (*Config, error) {
	var cfg Config
	if err := s.readJSON(configRefPrefix, &cfg); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotInitialized
		}
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) Create(in CreateInput) (*Bead, error) {
	cfg, err := s.Config()
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	now := time.Now().UTC()
	bead := &Bead{
		ID:           s.nextID(cfg.Prefix),
		Title:        title,
		Description:  strings.TrimSpace(in.Description),
		Type:         normalizedType(in.Type),
		Status:       "open",
		Priority:     normalizedPriority(in.Priority),
		Labels:       normalizeStrings(in.Labels),
		Dependencies: normalizeStrings(in.Dependencies),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.writeBead(bead); err != nil {
		return nil, err
	}
	return bead, nil
}

func (s *Store) Get(id string) (*Bead, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	var bead Bead
	if err := s.readJSON(issueRef(id), &bead); err != nil {
		return nil, err
	}
	return &bead, nil
}

func (s *Store) Update(id string, in UpdateInput) (*Bead, error) {
	bead, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return nil, fmt.Errorf("title is required")
		}
		bead.Title = title
	}
	if in.Description != nil {
		bead.Description = strings.TrimSpace(*in.Description)
	}
	if in.Type != nil {
		bead.Type = normalizedType(*in.Type)
	}
	if in.Priority != nil {
		bead.Priority = normalizedPriority(*in.Priority)
	}
	if in.Labels != nil {
		bead.Labels = normalizeStrings(in.Labels)
	}
	if len(in.AddLabels) > 0 || len(in.RemoveLabels) > 0 {
		labelSet := make(map[string]bool, len(bead.Labels))
		for _, label := range bead.Labels {
			labelSet[label] = true
		}
		for _, label := range normalizeStrings(in.AddLabels) {
			labelSet[label] = true
		}
		for _, label := range normalizeStrings(in.RemoveLabels) {
			delete(labelSet, label)
		}
		bead.Labels = make([]string, 0, len(labelSet))
		for label := range labelSet {
			bead.Labels = append(bead.Labels, label)
		}
		slices.Sort(bead.Labels)
	}
	bead.UpdatedAt = time.Now().UTC()
	if err := s.writeBead(bead); err != nil {
		return nil, err
	}
	return bead, nil
}

func (s *Store) Close(id, reason string) (*Bead, error) {
	bead, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	bead.Status = "closed"
	bead.CloseReason = strings.TrimSpace(reason)
	now := time.Now().UTC()
	bead.ClosedAt = &now
	bead.UpdatedAt = now
	if err := s.writeBead(bead); err != nil {
		return nil, err
	}
	return bead, nil
}

func (s *Store) Reopen(id string) (*Bead, error) {
	bead, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	bead.Status = "open"
	bead.CloseReason = ""
	bead.ClosedAt = nil
	bead.UpdatedAt = time.Now().UTC()
	if err := s.writeBead(bead); err != nil {
		return nil, err
	}
	return bead, nil
}

func (s *Store) AddLabels(id string, labels []string) (*Bead, error) {
	return s.Update(id, UpdateInput{AddLabels: labels})
}

func (s *Store) RemoveLabels(id string, labels []string) (*Bead, error) {
	return s.Update(id, UpdateInput{RemoveLabels: labels})
}

func (s *Store) SetDependencies(id string, deps []string) (*Bead, error) {
	bead, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	bead.Dependencies = normalizeStrings(deps)
	bead.UpdatedAt = time.Now().UTC()
	if err := s.writeBead(bead); err != nil {
		return nil, err
	}
	return bead, nil
}

func (s *Store) AddDependency(id, dep string) (*Bead, error) {
	bead, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	dep = strings.TrimSpace(dep)
	if dep == "" {
		return nil, fmt.Errorf("dependency id is required")
	}
	if bead.ID == dep {
		return nil, fmt.Errorf("cannot depend on itself")
	}
	_, err = s.Get(dep)
	if err != nil {
		return nil, fmt.Errorf("dependency %q: %w", dep, err)
	}
	bead.Dependencies = normalizeStrings(append(bead.Dependencies, dep))
	bead.UpdatedAt = time.Now().UTC()
	if err := s.writeBead(bead); err != nil {
		return nil, err
	}
	return bead, nil
}

func (s *Store) RemoveDependency(id, dep string) (*Bead, error) {
	bead, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	filtered := bead.Dependencies[:0]
	for _, existing := range bead.Dependencies {
		if existing != dep {
			filtered = append(filtered, existing)
		}
	}
	bead.Dependencies = normalizeStrings(filtered)
	bead.UpdatedAt = time.Now().UTC()
	if err := s.writeBead(bead); err != nil {
		return nil, err
	}
	return bead, nil
}

func (s *Store) List(filter Filter) ([]*Bead, error) {
	refs, err := s.issueRefs()
	if err != nil {
		return nil, err
	}
	issues := make([]*Bead, 0, len(refs))
	for _, ref := range refs {
		id := strings.TrimPrefix(ref, issueRefPrefix)
		bead, err := s.Get(id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		if matchFilter(bead, filter) {
			issues = append(issues, bead)
		}
	}
	slices.SortFunc(issues, func(a, b *Bead) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		return strings.Compare(a.ID, b.ID)
	})
	return issues, nil
}

func (s *Store) Search(query string, filter Filter) ([]*Bead, error) {
	filter.Query = query
	return s.List(filter)
}

func (s *Store) Ready() ([]*Bead, error) {
	all, err := s.List(Filter{All: true})
	if err != nil {
		return nil, err
	}
	status := make(map[string]string, len(all))
	for _, bead := range all {
		status[bead.ID] = bead.Status
	}
	var ready []*Bead
	for _, bead := range all {
		if bead.Status != "open" {
			continue
		}
		if !isBlocked(bead, status) {
			ready = append(ready, bead)
		}
	}
	return ready, nil
}

func (s *Store) Blocked() ([]*Bead, error) {
	all, err := s.List(Filter{All: true})
	if err != nil {
		return nil, err
	}
	status := make(map[string]string, len(all))
	for _, bead := range all {
		status[bead.ID] = bead.Status
	}
	var blocked []*Bead
	for _, bead := range all {
		if bead.Status != "open" {
			continue
		}
		if isBlocked(bead, status) {
			blocked = append(blocked, bead)
		}
	}
	return blocked, nil
}

func (s *Store) Export() ([]*Bead, error) {
	return s.List(Filter{All: true})
}

func isBlocked(bead *Bead, status map[string]string) bool {
	for _, dep := range bead.Dependencies {
		if status[dep] != "closed" {
			return true
		}
	}
	return false
}

func matchFilter(bead *Bead, filter Filter) bool {
	if filter.Status != "" && bead.Status != filter.Status {
		return false
	}
	if filter.Type != "" && bead.Type != normalizedType(filter.Type) {
		return false
	}
	if filter.Label != "" && !slices.Contains(bead.Labels, strings.TrimSpace(filter.Label)) {
		return false
	}
	if !filter.All && filter.Status == "" && bead.Status == "closed" {
		return false
	}
	if filter.Query == "" {
		return true
	}
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	if query == "" {
		return true
	}
	fields := []string{bead.ID, bead.Title, bead.Description, bead.Type, bead.Status, strings.Join(bead.Labels, " ")}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func (s *Store) writeBead(bead *Bead) error {
	bead.Labels = normalizeStrings(bead.Labels)
	bead.Dependencies = normalizeStrings(bead.Dependencies)
	return s.writeJSON(issueRef(bead.ID), bead)
}

func (s *Store) issueRefs() ([]string, error) {
	out, err := s.git("for-each-ref", "--format=%(refname)", issueRefPrefix)
	if err != nil {
		return nil, err
	}
	var refs []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			refs = append(refs, line)
		}
	}
	return refs, nil
}

func (s *Store) nextID(prefix string) string {
	for {
		var buf [4]byte
		if _, err := rand.Read(buf[:]); err != nil {
			panic(err)
		}
		id := fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(buf[:])[:6])
		if _, err := s.Get(id); errors.Is(err, ErrNotFound) {
			return id
		}
	}
}

func (s *Store) readJSON(ref string, v any) error {
	out, err := s.git("notes", "--ref="+trimNotesPrefix(ref), "show", s.anchor)
	if err != nil {
		if isMissingNote(err) {
			return ErrNotFound
		}
		return err
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return fmt.Errorf("decode %s: %w", ref, err)
	}
	return nil
}

func (s *Store) writeJSON(ref string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.gitInput(data, "notes", "--ref="+trimNotesPrefix(ref), "add", "-f", "-F", "-", s.anchor)
	return err
}

func (s *Store) git(args ...string) (string, error) {
	return s.gitInput(nil, args...)
}

func (s *Store) gitInput(input []byte, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.repo
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func resolveRepo(repo string) (string, error) {
	if repo == "" {
		repo = "."
	}
	cmd := exec.Command("git", "-C", repo, "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNotGitRepo, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func ensureAnchor(repo string) (string, error) {
	cmd := exec.Command("git", "-C", repo, "hash-object", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(anchorPayload)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create anchor object: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func normalizedType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if t == "" {
		return "task"
	}
	return t
}

func normalizedPriority(p int) int {
	if p < 0 {
		return 0
	}
	if p > 4 {
		return 4
	}
	return p
}

func normalizeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		for _, field := range strings.Split(item, ",") {
			field = strings.TrimSpace(field)
			if field == "" || seen[field] {
				continue
			}
			seen[field] = true
			out = append(out, field)
		}
	}
	slices.Sort(out)
	return out
}

func issueRef(id string) string {
	return issueRefPrefix + strings.TrimSpace(id)
}

func trimNotesPrefix(ref string) string {
	return strings.TrimPrefix(ref, "refs/notes/")
}

func isMissingNote(err error) bool {
	return strings.Contains(err.Error(), "no note found") || strings.Contains(err.Error(), "cannot read note data")
}

func DefaultPrefixFromRepo(repo string) string {
	base := strings.ToLower(filepath.Base(repo))
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == string(os.PathSeparator) {
		return "bead"
	}
	return base
}
