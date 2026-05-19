package policies

import (
	"testing"
)

func TestDefaultEngine_no_policy_denies(t *testing.T) {
	eng := NewDefaultEngine()
	res := eng.Evaluate(Request{Subject: "u", Action: "read", Resource: "r"})
	if res.Allowed {
		t.Error("expected deny when no policy")
	}
	if res.Reason != "no matching policy" {
		t.Errorf("Reason = %q", res.Reason)
	}
}

func TestDefaultEngine_first_deny_wins(t *testing.T) {
	eng := NewDefaultEngine()
	eng.AddPolicy(NewStaticPolicy("allow", EffectAllow, "u", "read", "r"))
	eng.AddPolicy(NewStaticPolicy("deny", EffectDeny, "u", "read", "r"))
	res := eng.Evaluate(Request{Subject: "u", Action: "read", Resource: "r"})
	if res.Allowed {
		t.Error("expected deny (first deny wins)")
	}
	if res.Effect != EffectDeny {
		t.Errorf("Effect = %v", res.Effect)
	}
}

func TestDefaultEngine_allow_when_no_deny(t *testing.T) {
	eng := NewDefaultEngine()
	eng.AddPolicy(NewStaticPolicy("allow", EffectAllow, "u", "read", "r"))
	res := eng.Evaluate(Request{Subject: "u", Action: "read", Resource: "r"})
	if !res.Allowed {
		t.Errorf("result = %+v", res)
	}
}

func TestDefaultEngine_RemovePolicy(t *testing.T) {
	eng := NewDefaultEngine()
	eng.AddPolicy(NewStaticPolicy("p1", EffectAllow, "u", "read", "r"))
	eng.RemovePolicy("p1")
	res := eng.Evaluate(Request{Subject: "u", Action: "read", Resource: "r"})
	if res.Allowed {
		t.Error("expected deny after removing only policy")
	}
}
