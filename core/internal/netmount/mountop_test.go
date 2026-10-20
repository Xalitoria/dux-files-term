package netmount

import (
	"sync"
	"testing"
	"time"

	"github.com/duxlabs-dev/dux-go/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recorder struct {
	mu     sync.Mutex
	asks   []Ask
	notify chan Ask
}

func (r *recorder) Publish(_ string, data any) {
	ask, ok := data.(Ask)
	if !ok {
		return
	}
	r.mu.Lock()
	r.asks = append(r.asks, ask)
	r.mu.Unlock()
	r.notify <- ask
}

func opService(t *testing.T) (*Service, *recorder) {
	t.Helper()
	bus := &recorder{notify: make(chan Ask, 4)}
	svc := NewService(bus, t.TempDir())
	return svc, bus
}

func TestAskPasswordPublishesTheQuestionAndWaitsForTheAnswer(t *testing.T) {
	svc, bus := opService(t)
	op := svc.newOp("ftp://host/")
	svc.trackOp(op)

	var handled, aborted, anonymous bool
	var password, user, domain string
	var save uint32
	done := make(chan struct{})

	go func() {
		handled, aborted, password, user, domain, anonymous, save, _ = op.AskPassword("Enter password", "ana", "WORK", needPassword|needUsername|savingAllowed)
		close(done)
	}()

	ask := <-bus.notify
	assert.Equal(t, "ask", ask.Kind)
	assert.Equal(t, "password", ask.Type)
	assert.Equal(t, "ftp://host/", ask.URI)
	assert.Equal(t, "ana", ask.DefaultUser)
	assert.Equal(t, "WORK", ask.DefaultDomain)
	assert.True(t, ask.NeedPassword)
	assert.True(t, ask.NeedUsername)
	assert.True(t, ask.CanSave)
	assert.False(t, ask.NeedDomain)
	assert.False(t, ask.CanAnonymous)

	require.NoError(t, svc.Answer(ask.ID, Answer{Password: "hunter2", User: "ana", Save: 2}))
	<-done

	assert.True(t, handled)
	assert.False(t, aborted)
	assert.Equal(t, "hunter2", password)
	assert.Equal(t, "ana", user)
	assert.Empty(t, domain)
	assert.False(t, anonymous)
	assert.Equal(t, uint32(2), save)
}

func TestAskPasswordReportsAnAbort(t *testing.T) {
	svc, bus := opService(t)
	op := svc.newOp("ftp://host/")
	svc.trackOp(op)

	result := make(chan bool, 1)
	go func() {
		_, aborted, _, _, _, _, _, _ := op.AskPassword("Enter password", "", "", needPassword)
		result <- aborted
	}()

	ask := <-bus.notify
	require.NoError(t, svc.Answer(ask.ID, Answer{Abort: true}))
	assert.True(t, <-result)
}

func TestAskQuestionCarriesTheChoices(t *testing.T) {
	svc, bus := opService(t)
	op := svc.newOp("sftp://host/")
	svc.trackOp(op)

	choice := make(chan uint32, 1)
	go func() {
		_, _, picked, _ := op.AskQuestion("Unknown host key", []string{"Log in anyway", "Cancel"})
		choice <- picked
	}()

	ask := <-bus.notify
	assert.Equal(t, "question", ask.Type)
	assert.Equal(t, []string{"Log in anyway", "Cancel"}, ask.Choices)

	require.NoError(t, svc.Answer(ask.ID, Answer{Choice: 1}))
	assert.Equal(t, uint32(1), <-choice)
}

func TestAskGivesUpWhenNobodyAnswers(t *testing.T) {
	previous := askTimeout
	askTimeout = 20 * time.Millisecond
	t.Cleanup(func() { askTimeout = previous })

	svc, _ := opService(t)
	op := svc.newOp("ftp://host/")
	svc.trackOp(op)

	_, aborted, _, _, _, _, _, _ := op.AskPassword("Enter password", "", "", needPassword)
	assert.True(t, aborted)
}

func TestAnswerRejectsQuestionsThatAreGone(t *testing.T) {
	svc, _ := opService(t)
	op := svc.newOp("ftp://host/")
	svc.trackOp(op)

	err := svc.Answer(string(op.path)+"/7", Answer{})
	assert.Equal(t, files.CodeInvalid, files.CodeOf(err))

	err = svc.Answer("nonsense", Answer{})
	assert.Equal(t, files.CodeInvalid, files.CodeOf(err))
}

func TestReleaseAbortsEverythingStillWaiting(t *testing.T) {
	svc, bus := opService(t)
	op := svc.newOp("ftp://host/")
	svc.trackOp(op)

	result := make(chan bool, 1)
	go func() {
		_, aborted, _, _, _, _, _, _ := op.AskPassword("Enter password", "", "", needPassword)
		result <- aborted
	}()

	<-bus.notify
	op.release(nil)
	assert.True(t, <-result)

	_, aborted, _, _, _, _, _, _ := op.AskPassword("Enter password", "", "", needPassword)
	assert.True(t, aborted, "a released operation answers without asking again")
}

func TestAskPasswordUsesAPasswordFromTheURIOnce(t *testing.T) {
	svc, bus := opService(t)
	op := svc.newOp("sftp://ana@host/")
	op.password = "s3cret"
	svc.trackOp(op)

	handled, aborted, password, user, domain, anonymous, save, _ := op.AskPassword("Enter password", "ana", "WORK", needPassword|savingAllowed)
	assert.True(t, handled)
	assert.False(t, aborted)
	assert.Equal(t, "s3cret", password)
	assert.Equal(t, "ana", user)
	assert.Equal(t, "WORK", domain)
	assert.False(t, anonymous)
	assert.Zero(t, save)
	assert.Empty(t, bus.notify)

	go func() { _, _, _, _, _, _, _, _ = op.AskPassword("Wrong password", "ana", "", needPassword) }()
	ask := <-bus.notify
	assert.Equal(t, "Wrong password", ask.Message)
	assert.Equal(t, "sftp://ana@host/", ask.URI)
	require.NoError(t, svc.Answer(ask.ID, Answer{Abort: true}))
}

func TestAskPasswordAsksWhenTheURIHadNoUser(t *testing.T) {
	svc, bus := opService(t)
	op := svc.newOp("sftp://host/")
	op.password = "s3cret"
	svc.trackOp(op)

	go func() { _, _, _, _, _, _, _, _ = op.AskPassword("Enter password", "", "", needPassword|needUsername) }()
	ask := <-bus.notify
	assert.True(t, ask.NeedUsername)
	require.NoError(t, svc.Answer(ask.ID, Answer{Abort: true}))
}
