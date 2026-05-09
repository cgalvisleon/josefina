package stmt

import "testing"

func TestSetCache_Basic(t *testing.T) {
	stmts, err := ParseText(`SET CACHE mykey myvalue`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SetCacheStmt)
	if s.Key != "mykey" || s.Value != "myvalue" {
		t.Errorf("unexpected key/value: %q %q", s.Key, s.Value)
	}
	if s.Duration != 0 {
		t.Errorf("expected zero duration, got %v", s.Duration)
	}
}

func TestSetCache_WithDuration(t *testing.T) {
	stmts, err := ParseText(`SET CACHE tok abc123 300`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(SetCacheStmt)
	if s.Key != "tok" || s.Value != "abc123" {
		t.Errorf("unexpected key/value: %q %q", s.Key, s.Value)
	}
	if s.Duration != 300 {
		t.Errorf("expected 300, got %v", s.Duration)
	}
}

func TestGetCache_Basic(t *testing.T) {
	stmts, err := ParseText(`GET CACHE mykey`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(GetCacheStmt)
	if s.Key != "mykey" {
		t.Errorf("unexpected key: %q", s.Key)
	}
}

func TestDelCache_Basic(t *testing.T) {
	stmts, err := ParseText(`DEL CACHE mykey`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(DelCacheStmt)
	if s.Key != "mykey" {
		t.Errorf("unexpected key: %q", s.Key)
	}
}

func TestExistCache_Basic(t *testing.T) {
	stmts, err := ParseText(`EXIST CACHE mykey`)
	if err != nil {
		t.Fatal(err)
	}
	s := stmts[0].(ExistCacheStmt)
	if s.Key != "mykey" {
		t.Errorf("unexpected key: %q", s.Key)
	}
}
