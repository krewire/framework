// Package vdom provides the virtual DOM tree shared by server-side HTML
// rendering and client-side reconciliation (KWF-T4X9P).
//
// A VNode is an immutable description: build trees with El, Text, and
// Component, diff two trees with Diff, and apply patches either to real DOM
// (client slices, later) or to strings via RenderHTML (server).
//
// # Prop normalization rules
//
// Both Diff and RenderHTML funnel every Props map through NormalizeProps so
// the server output and the client patches can never disagree:
//
//   - Values are used verbatim except for "class", whose space-separated
//     tokens are trimmed and re-joined with single spaces.
//   - An attribute whose value is "" renders as a bare boolean attribute if
//     its key is in BoolAttrs (e.g. <input disabled>), and is dropped
//     otherwise.
//   - data-* and aria-* keys pass through untouched.
//
// Because missing and "" are equivalent for BoolAttrs members, removing such
// a key produces no patch.
//
// # Patch application contract
//
// Each Patch locates its parent by Path (indices from the diffed root).
// PatchRemove carries an OLD-layout child Index and is emitted first,
// descending, so sequential application on the existing tree is safe.
// All other patches reference positions in the NEXT tree: property/text
// updates come next, then inserts in ascending order. A client applier walks
// its own mirrored VNode structure using these indices; server consumers can
// ignore patches entirely and call RenderHTML.
package vdom
