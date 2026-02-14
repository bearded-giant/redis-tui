package types

import "testing"

func TestNewTreeNode(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		fullPath string
		isKey    bool
	}{
		{"leaf node", "mykey", "user:mykey", true},
		{"folder node", "user", "user", false},
		{"empty path", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewTreeNode(tt.nodeName, tt.fullPath, tt.isKey)
			if node.Name != tt.nodeName {
				t.Errorf("Name = %q, want %q", node.Name, tt.nodeName)
			}
			if node.FullPath != tt.fullPath {
				t.Errorf("FullPath = %q, want %q", node.FullPath, tt.fullPath)
			}
			if node.IsKey != tt.isKey {
				t.Errorf("IsKey = %v, want %v", node.IsKey, tt.isKey)
			}
			if node.Children == nil {
				t.Error("Children should be initialized as empty slice, got nil")
			}
			if len(node.Children) != 0 {
				t.Errorf("Children should be empty, got %d", len(node.Children))
			}
		})
	}
}

func TestAddChild(t *testing.T) {
	t.Run("single child", func(t *testing.T) {
		parent := NewTreeNode("root", "root", false)
		child := NewTreeNode("child1", "root:child1", true)
		parent.AddChild(child)

		if parent.ChildCount != 1 {
			t.Errorf("ChildCount = %d, want 1", parent.ChildCount)
		}
		if len(parent.Children) != 1 {
			t.Errorf("len(Children) = %d, want 1", len(parent.Children))
		}
		if parent.Children[0] != child {
			t.Error("child not found in children")
		}
	})

	t.Run("multiple children", func(t *testing.T) {
		parent := NewTreeNode("root", "root", false)
		parent.AddChild(NewTreeNode("a", "root:a", true))
		parent.AddChild(NewTreeNode("b", "root:b", true))
		parent.AddChild(NewTreeNode("c", "root:c", true))

		if parent.ChildCount != 3 {
			t.Errorf("ChildCount = %d, want 3", parent.ChildCount)
		}
	})

	t.Run("nested children", func(t *testing.T) {
		root := NewTreeNode("root", "root", false)
		child := NewTreeNode("child", "root:child", false)
		grandchild := NewTreeNode("grandchild", "root:child:grandchild", true)

		root.AddChild(child)
		child.AddChild(grandchild)

		if root.ChildCount != 1 {
			t.Errorf("root ChildCount = %d, want 1", root.ChildCount)
		}
		if child.ChildCount != 1 {
			t.Errorf("child ChildCount = %d, want 1", child.ChildCount)
		}
	})
}

func TestFindChild(t *testing.T) {
	parent := NewTreeNode("root", "root", false)
	child1 := NewTreeNode("alpha", "root:alpha", true)
	child2 := NewTreeNode("beta", "root:beta", true)
	child3 := NewTreeNode("gamma", "root:gamma", true)
	parent.AddChild(child1)
	parent.AddChild(child2)
	parent.AddChild(child3)

	t.Run("existing child", func(t *testing.T) {
		found := parent.FindChild("beta")
		if found == nil {
			t.Fatal("expected to find child 'beta'")
		}
		if found != child2 {
			t.Error("found wrong child")
		}
	})

	t.Run("missing child", func(t *testing.T) {
		found := parent.FindChild("missing")
		if found != nil {
			t.Errorf("expected nil for missing child, got %v", found)
		}
	})

	t.Run("first child", func(t *testing.T) {
		found := parent.FindChild("alpha")
		if found != child1 {
			t.Error("expected to find first child")
		}
	})
}

func TestToggle(t *testing.T) {
	node := NewTreeNode("test", "test", false)

	t.Run("collapsed to expanded", func(t *testing.T) {
		if node.Expanded {
			t.Error("should start collapsed")
		}
		node.Toggle()
		if !node.Expanded {
			t.Error("should be expanded after toggle")
		}
	})

	t.Run("expanded to collapsed", func(t *testing.T) {
		node.Toggle()
		if node.Expanded {
			t.Error("should be collapsed after second toggle")
		}
	})
}

func TestGetDepth(t *testing.T) {
	tests := []struct {
		name     string
		fullPath string
		expected int
	}{
		{"no colons", "simple", 0},
		{"one colon", "user:123", 1},
		{"three colons", "a:b:c:d", 3},
		{"empty path", "", 0},
		{"trailing colon", "user:", 1},
		{"leading colon", ":user", 1},
		{"multiple nested", "app:module:feature:key:subkey", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewTreeNode("test", tt.fullPath, true)
			got := node.GetDepth()
			if got != tt.expected {
				t.Errorf("GetDepth() for path %q = %d, want %d", tt.fullPath, got, tt.expected)
			}
		})
	}
}
