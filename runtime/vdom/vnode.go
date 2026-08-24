package vdom

// NodeKind discriminates what a VNode hosts.
type NodeKind uint8

const (
	// KindElement is a plain DOM element with Tag, optional Props and Children.
	KindElement NodeKind = iota
	// KindText is a text node; only Text is meaningful.
	KindText
	// KindComponent is a hosted component placeholder resolved by name during
	// SSR substitution or client hydration.
	KindComponent
)

// VNode is an immutable virtual node. Build trees via El, Text and Component.
type VNode struct {
	Kind          NodeKind
	Tag           string
	Text          string
	Props         map[string]string
	Children      []*VNode
	Key           string
	ComponentName string
}

// El returns an element node.
func El(tag string, props map[string]string, children ...*VNode) *VNode {
	return &VNode{Kind: KindElement, Tag: tag, Props: props, Children: children}
}

// Text returns a text node.
func Text(s string) *VNode {
	return &VNode{Kind: KindText, Text: s}
}

// Component returns a hosted-component placeholder resolved by name. The key
// participates in keyed reconciliation just like an element key.
func Component(name, key string, props map[string]string) *VNode {
	return &VNode{Kind: KindComponent, ComponentName: name, Key: key, Props: props}
}
