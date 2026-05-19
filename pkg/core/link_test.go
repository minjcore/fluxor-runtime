package core

import (
	"context"
	"testing"
	"time"
)

func TestNewLinkManager(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	m := NewLinkManager(eb)
	if m == nil {
		t.Fatal("NewLinkManager returned nil")
	}
}

func TestLinkManager_Link_Unlink(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	m := NewLinkManager(gocmd.EventBus())

	err := m.Link("", "target")
	if err == nil {
		t.Error("Link with empty selfID should fail")
	}
	err = m.Link("self", "")
	if err == nil {
		t.Error("Link with empty targetID should fail")
	}
	err = m.Link("self", "self")
	if err == nil {
		t.Error("Link to self should fail")
	}

	err = m.Link("a", "b")
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	err = m.Unlink("a", "b")
	if err != nil {
		t.Fatalf("Unlink: %v", err)
	}
}

func TestLinkManager_Monitor_Demonitor(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	m := NewLinkManager(gocmd.EventBus())

	_, err := m.Monitor("", "target")
	if err == nil {
		t.Error("Monitor with empty selfID should fail")
	}
	_, err = m.Monitor("self", "")
	if err == nil {
		t.Error("Monitor with empty targetID should fail")
	}

	ref, err := m.Monitor("monitorer", "target")
	if err != nil {
		t.Fatalf("Monitor: %v", err)
	}
	err = m.Demonitor(ref)
	if err != nil {
		t.Fatalf("Demonitor: %v", err)
	}
	err = m.Demonitor(ref)
	if err != nil {
		t.Fatalf("Demonitor again: %v", err)
	}
}

func TestLinkManager_NotifyExit_DeliversToLinked(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()
	m := NewLinkManager(eb)

	err := m.Link("a", "b")
	if err != nil {
		t.Fatalf("Link: %v", err)
	}

	received := make(chan *ExitSignal, 2)
	consumer := eb.Consumer(ExitSignalAddress("b"))
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		var sig ExitSignal
		if e := msg.DecodeBody(&sig); e != nil {
			return e
		}
		received <- &sig
		return nil
	})

	m.NotifyExit("a", StateRunning, nil)

	select {
	case sig := <-received:
		if sig.From != "a" {
			t.Errorf("From = %q, want a", sig.From)
		}
		if sig.State != StateRunning {
			t.Errorf("State = %v, want StateRunning", sig.State)
		}
	case <-time.After(time.Second):
		t.Fatal("linked peer did not receive exit signal")
	}
}

func TestLinkManager_NotifyExit_DeliversToMonitorer(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()
	m := NewLinkManager(eb)

	_, err := m.Monitor("monitorer", "target")
	if err != nil {
		t.Fatalf("Monitor: %v", err)
	}

	received := make(chan *ExitSignal, 1)
	consumer := eb.Consumer(ExitSignalAddress("monitorer"))
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		var sig ExitSignal
		if e := msg.DecodeBody(&sig); e != nil {
			return e
		}
		received <- &sig
		return nil
	})

	m.NotifyExit("target", StateFailed, nil)

	select {
	case sig := <-received:
		if sig.From != "target" {
			t.Errorf("From = %q, want target", sig.From)
		}
		if sig.State != StateFailed {
			t.Errorf("State = %v, want StateFailed", sig.State)
		}
	case <-time.After(time.Second):
		t.Fatal("monitorer did not receive exit signal")
	}
}

func TestExitSignalAddress(t *testing.T) {
	addr := ExitSignalAddress("deploy-1")
	if addr != "core.exit.deploy-1" {
		t.Errorf("ExitSignalAddress = %q, want core.exit.deploy-1", addr)
	}
}
