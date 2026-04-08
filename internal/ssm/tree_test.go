package ssm

import (
	"testing"
	"time"
)

func TestNodeType_String(t *testing.T) {
	tests := []struct {
		nt   NodeType
		want string
	}{
		{NodeFolder, "folder"},
		{NodeParameter, "parameter"},
		{NodeType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.nt.String(); got != tt.want {
			t.Errorf("NodeType(%d).String() = %q, want %q", tt.nt, got, tt.want)
		}
	}
}

func TestNewFolder(t *testing.T) {
	n := NewFolder("config", "/app/config")
	if n.Name != "config" {
		t.Errorf("Name = %q, want %q", n.Name, "config")
	}
	if n.Path != "/app/config" {
		t.Errorf("Path = %q, want %q", n.Path, "/app/config")
	}
	if n.Type != NodeFolder {
		t.Errorf("Type = %v, want NodeFolder", n.Type)
	}
	if n.Children == nil {
		t.Error("Children should be initialized, got nil")
	}
	if n.Meta != nil {
		t.Error("Meta should be nil for folder")
	}
}

func TestNewParameter(t *testing.T) {
	meta := &ParameterMeta{
		Type:         "SecureString",
		Version:      3,
		LastModified: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		ARN:          "arn:aws:ssm:us-east-1:123456789:parameter/app/db_pass",
		DataType:     "text",
	}
	n := NewParameter("db_pass", "/app/db_pass", meta)
	if n.Name != "db_pass" {
		t.Errorf("Name = %q, want %q", n.Name, "db_pass")
	}
	if n.Type != NodeParameter {
		t.Errorf("Type = %v, want NodeParameter", n.Type)
	}
	if n.Children != nil {
		t.Error("Children should be nil for parameter")
	}
	if n.Meta == nil || n.Meta.Type != "SecureString" {
		t.Errorf("Meta.Type = %v, want SecureString", n.Meta)
	}
}

func TestIsFolder_IsParameter(t *testing.T) {
	folder := NewFolder("f", "/f")
	param := NewParameter("p", "/p", nil)

	if !folder.IsFolder() {
		t.Error("folder.IsFolder() should be true")
	}
	if folder.IsParameter() {
		t.Error("folder.IsParameter() should be false")
	}
	if param.IsFolder() {
		t.Error("param.IsFolder() should be false")
	}
	if !param.IsParameter() {
		t.Error("param.IsParameter() should be true")
	}
}

func TestIsSecureString(t *testing.T) {
	secure := NewParameter("s", "/s", &ParameterMeta{Type: "SecureString"})
	plain := NewParameter("p", "/p", &ParameterMeta{Type: "String"})
	noMeta := NewParameter("n", "/n", nil)
	folder := NewFolder("f", "/f")

	if !secure.IsSecureString() {
		t.Error("secure param should return true")
	}
	if plain.IsSecureString() {
		t.Error("plain param should return false")
	}
	if noMeta.IsSecureString() {
		t.Error("no-meta param should return false")
	}
	if folder.IsSecureString() {
		t.Error("folder should return false")
	}
}

func TestFindChild(t *testing.T) {
	root := NewFolder("/", "/")
	child := NewFolder("app", "/app")
	root.AddChild(child)

	found := root.FindChild("app")
	if found != child {
		t.Errorf("FindChild(\"app\") = %v, want %v", found, child)
	}
	if root.FindChild("missing") != nil {
		t.Error("FindChild(\"missing\") should return nil")
	}
}

func TestChildCount(t *testing.T) {
	root := NewFolder("/", "/")
	if root.ChildCount() != 0 {
		t.Errorf("ChildCount() = %d, want 0", root.ChildCount())
	}
	root.AddChild(NewFolder("a", "/a"))
	root.AddChild(NewParameter("b", "/b", nil))
	if root.ChildCount() != 2 {
		t.Errorf("ChildCount() = %d, want 2", root.ChildCount())
	}
}

func TestParameterCount(t *testing.T) {
	root := NewFolder("/", "/")
	app := NewFolder("app", "/app")
	app.AddChild(NewParameter("key1", "/app/key1", nil))
	app.AddChild(NewParameter("key2", "/app/key2", nil))
	root.AddChild(app)
	root.AddChild(NewParameter("top", "/top", nil))

	if got := root.ParameterCount(); got != 3 {
		t.Errorf("ParameterCount() = %d, want 3", got)
	}
	if got := app.ParameterCount(); got != 2 {
		t.Errorf("app.ParameterCount() = %d, want 2", got)
	}
}

func TestSortChildren(t *testing.T) {
	root := NewFolder("/", "/")
	root.AddChild(NewParameter("zebra", "/zebra", nil))
	root.AddChild(NewFolder("beta", "/beta"))
	root.AddChild(NewParameter("alpha", "/alpha", nil))
	root.AddChild(NewFolder("alpha_dir", "/alpha_dir"))

	root.SortChildren()

	// Expect folders first (sorted), then parameters (sorted).
	expected := []struct {
		name string
		typ  NodeType
	}{
		{"alpha_dir", NodeFolder},
		{"beta", NodeFolder},
		{"alpha", NodeParameter},
		{"zebra", NodeParameter},
	}

	if len(root.Children) != len(expected) {
		t.Fatalf("got %d children, want %d", len(root.Children), len(expected))
	}
	for i, e := range expected {
		c := root.Children[i]
		if c.Name != e.name || c.Type != e.typ {
			t.Errorf("child[%d] = {%q, %v}, want {%q, %v}", i, c.Name, c.Type, e.name, e.typ)
		}
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/app/config/db_host", []string{"app", "config", "db_host"}},
		{"/single", []string{"single"}},
		{"/", nil},
		{"", nil},
		{"/a//b/", []string{"a", "b"}},
	}
	for _, tt := range tests {
		got := splitPath(tt.path)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("splitPath(%q) = %v, want %v", tt.path, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
			}
		}
	}
}

func TestBuildTree(t *testing.T) {
	params := []FlatParam{
		{Path: "/app/config/db_host", Meta: &ParameterMeta{Type: "String", Version: 1}},
		{Path: "/app/config/db_pass", Meta: &ParameterMeta{Type: "SecureString", Version: 2}},
		{Path: "/app/api_key", Meta: &ParameterMeta{Type: "String", Version: 1}},
		{Path: "/infra/vpc_id", Meta: &ParameterMeta{Type: "String", Version: 1}},
	}

	root := BuildTree(params)

	// Root should be "/"
	if root.Name != "/" || root.Path != "/" {
		t.Errorf("root = {%q, %q}, want {\"/\", \"/\"}", root.Name, root.Path)
	}
	if !root.IsFolder() {
		t.Error("root should be a folder")
	}

	// Root should have 2 folder children: app, infra (sorted)
	if root.ChildCount() != 2 {
		t.Fatalf("root.ChildCount() = %d, want 2", root.ChildCount())
	}
	if root.Children[0].Name != "app" {
		t.Errorf("root.Children[0].Name = %q, want \"app\"", root.Children[0].Name)
	}
	if root.Children[1].Name != "infra" {
		t.Errorf("root.Children[1].Name = %q, want \"infra\"", root.Children[1].Name)
	}

	// app should have: folder "config" then parameter "api_key"
	app := root.Children[0]
	if app.ChildCount() != 2 {
		t.Fatalf("app.ChildCount() = %d, want 2", app.ChildCount())
	}
	// After sort: folder "config" first, then parameter "api_key"
	if app.Children[0].Name != "config" || !app.Children[0].IsFolder() {
		t.Errorf("app child[0] = {%q, folder=%v}, want {\"config\", true}", app.Children[0].Name, app.Children[0].IsFolder())
	}
	if app.Children[1].Name != "api_key" || !app.Children[1].IsParameter() {
		t.Errorf("app child[1] = {%q, param=%v}, want {\"api_key\", true}", app.Children[1].Name, app.Children[1].IsParameter())
	}

	// config should have 2 parameters: db_host, db_pass
	config := app.Children[0]
	if config.ChildCount() != 2 {
		t.Fatalf("config.ChildCount() = %d, want 2", config.ChildCount())
	}
	if config.Children[0].Name != "db_host" {
		t.Errorf("config child[0] = %q, want \"db_host\"", config.Children[0].Name)
	}
	if config.Children[1].Name != "db_pass" {
		t.Errorf("config child[1] = %q, want \"db_pass\"", config.Children[1].Name)
	}

	// Verify metadata is preserved
	dbPass := config.Children[1]
	if !dbPass.IsSecureString() {
		t.Error("db_pass should be SecureString")
	}
	if dbPass.Meta.Version != 2 {
		t.Errorf("db_pass version = %d, want 2", dbPass.Meta.Version)
	}

	// Total parameter count
	if got := root.ParameterCount(); got != 4 {
		t.Errorf("root.ParameterCount() = %d, want 4", got)
	}
}

func TestBuildTree_Empty(t *testing.T) {
	root := BuildTree(nil)
	if root.ChildCount() != 0 {
		t.Errorf("empty tree should have 0 children, got %d", root.ChildCount())
	}
	if root.ParameterCount() != 0 {
		t.Errorf("empty tree should have 0 parameters, got %d", root.ParameterCount())
	}
}
