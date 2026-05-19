package concurrency

import (
	"context"
	"testing"
)

func TestNewBoundedMailbox(t *testing.T) {
	mailbox := NewBoundedMailbox(10)

	if mailbox == nil {
		t.Error("NewBoundedMailbox() should not return nil")
	}

	if mailbox.Capacity() != 10 {
		t.Errorf("Capacity() = %d, want 10", mailbox.Capacity())
	}
}

func TestMailbox_Send(t *testing.T) {
	mailbox := NewBoundedMailbox(2)

	// Test valid send
	err := mailbox.Send("message1")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Fill mailbox
	mailbox.Send("message2")

	// Test full mailbox (should return error)
	err = mailbox.Send("message3")
	if err != ErrMailboxFull {
		t.Errorf("Send() to full mailbox error = %v, want ErrMailboxFull", err)
	}
}

func TestMailbox_Receive(t *testing.T) {
	mailbox := NewBoundedMailbox(10)
	ctx := context.Background()

	// Send message
	mailbox.Send("test message")

	// Receive message
	msg, err := mailbox.Receive(ctx)
	if err != nil {
		t.Errorf("Receive() error = %v", err)
	}

	if msg != "test message" {
		t.Errorf("Receive() = %v, want test message", msg)
	}
}

func TestMailbox_TryReceive(t *testing.T) {
	mailbox := NewBoundedMailbox(10)

	// Test empty mailbox
	msg, ok, err := mailbox.TryReceive()
	if err != nil {
		t.Errorf("TryReceive() on empty mailbox error = %v", err)
	}
	if ok {
		t.Error("TryReceive() on empty mailbox should return ok=false")
	}
	if msg != nil {
		t.Errorf("TryReceive() on empty mailbox msg = %v, want nil", msg)
	}

	// Send and receive
	mailbox.Send("test")
	msg, ok, err = mailbox.TryReceive()
	if err != nil {
		t.Errorf("TryReceive() error = %v", err)
	}
	if !ok {
		t.Error("TryReceive() should return ok=true when message available")
	}
	if msg != "test" {
		t.Errorf("TryReceive() = %v, want test", msg)
	}
}

func TestMailbox_Close(t *testing.T) {
	mailbox := NewBoundedMailbox(10)

	mailbox.Close()

	if !mailbox.IsClosed() {
		t.Error("IsClosed() should return true after Close()")
	}

	// Test send after close
	err := mailbox.Send("test")
	if err != ErrMailboxClosed {
		t.Errorf("Send() after close error = %v, want ErrMailboxClosed", err)
	}

	// Test receive after close
	ctx := context.Background()
	_, err = mailbox.Receive(ctx)
	if err != ErrMailboxClosed {
		t.Errorf("Receive() after close error = %v, want ErrMailboxClosed", err)
	}
}

func TestMailbox_Size(t *testing.T) {
	mailbox := NewBoundedMailbox(10)

	if mailbox.Size() != 0 {
		t.Errorf("Size() = %d, want 0", mailbox.Size())
	}

	mailbox.Send("msg1")
	if mailbox.Size() != 1 {
		t.Errorf("Size() = %d, want 1", mailbox.Size())
	}

	mailbox.Send("msg2")
	if mailbox.Size() != 2 {
		t.Errorf("Size() = %d, want 2", mailbox.Size())
	}
}
