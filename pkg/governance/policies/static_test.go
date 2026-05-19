package policies

import (
	"testing"
)

func TestStaticPolicy_Evaluate_match(t *testing.T) {
	p := NewStaticPolicy("allow-admin", EffectAllow, "admin", "write", "users")
	res, applied := p.Evaluate(Request{Subject: "admin", Action: "write", Resource: "users"})
	if !applied {
		t.Fatal("expected policy to apply")
	}
	if !res.Allowed || res.Effect != EffectAllow {
		t.Errorf("result = %+v", res)
	}
}

func TestStaticPolicy_Evaluate_no_match(t *testing.T) {
	p := NewStaticPolicy("allow-admin", EffectAllow, "admin", "write", "users")
	_, applied := p.Evaluate(Request{Subject: "user", Action: "read", Resource: "files"})
	if applied {
		t.Error("expected policy not to apply")
	}
}

func TestStaticPolicy_Evaluate_empty_matches_any(t *testing.T) {
	p := NewStaticPolicy("deny-all-write", EffectDeny, "", "write", "")
	res, applied := p.Evaluate(Request{Subject: "any", Action: "write", Resource: "anything"})
	if !applied {
		t.Fatal("expected policy to apply")
	}
	if res.Allowed || res.Effect != EffectDeny {
		t.Errorf("result = %+v", res)
	}
}
