package prompt

import "testing"

func TestNormalizeNumberWords(t *testing.T) {
tests := []struct{in, want string}{
{"what is one plus one", "what is 1 + 1"},
{"What is One Plus One", "What is 1 + 1"},
{"twenty-one divided by seven", "21 / 7"},
{"five times three equals fifteen", "5 * 3 = 15"},
{"half plus quarter", "1/2 + 1/4"},
}
for _, tc := range tests {
got := NormalizeNumberWords(tc.in)
if got != tc.want {
t.Errorf("%q -> %q want %q", tc.in, got, tc.want)
}
}
SetNumberWords(false)
b := NewBuilder()
p := b.Build("what is one plus one", nil, "other")
if contains2(p, "1 + 1") {
t.Errorf("disabled Build should not contain normalized")
}
SetNumberWords(true)
b2 := NewBuilder()
p2 := b2.Build("what is one plus one", nil, "other")
if !contains2(p2, "1 + 1") {
t.Errorf("enabled Build should contain normalized")
}
SetNumberWords(false)
}
func contains2(s, sub string) bool {
if len(sub) > len(s) { return false }
for i:=0; i <= len(s)-len(sub); i++ {
if s[i:i+len(sub)]==sub { return true }
}
return false
}
