package imap

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/yesnault/go-imap/imap"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name for test imap
const Name = "imap"

var imapLogMask = imap.LogNone
var imapSafeLogMask = imap.LogNone

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	IMAPHost        string `json:"imaphost,omitempty" yaml:"imaphost,omitempty"`
	IMAPPort        string `json:"imapport,omitempty" yaml:"imapport,omitempty"`
	IMAPUser        string `json:"imapuser,omitempty" yaml:"imapuser,omitempty"`
	IMAPPassword    string `json:"imappassword,omitempty" yaml:"imappassword,omitempty"`
	MBox            string `json:"mbox,omitempty" yaml:"mbox,omitempty"`
	MBoxOnSuccess   string `json:"mboxonsuccess,omitempty" yaml:"mboxonsuccess,omitempty"`
	DeleteOnSuccess bool   `json:"deleteonsuccess,omitempty" yaml:"deleteonsuccess,omitempty"`
	SearchFrom      string `json:"searchfrom,omitempty" yaml:"searchfrom,omitempty"`
	SearchSubject   string `json:"searchsubject,omitempty" yaml:"searchsubject,omitempty"`
	SearchBody      string `json:"searchbody,omitempty" yaml:"searchbody,omitempty"`
}

// Mail contains an analyzed mail
type Mail struct {
	From    string
	Subject string
	UID     uint32
	Body    string
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	Err         string   `json:"error" yaml:"error"`
	Subject     string   `json:"subject,omitempty" yaml:"subject,omitempty"`
	Body        string   `json:"body,omitempty" yaml:"body,omitempty"`
	TimeSeconds float64  `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	TimeHuman   string   `json:"timeHuman,omitempty" yaml:"timeHuman,omitempty"`
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldNotExist"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx venom.TestCaseContext, l venom.Logger, step venom.TestStep) (venom.ExecutorResult, error) {
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}

	start := time.Now()

	result := Result{Executor: t}
	find, errs := t.getMail(l)
	if errs != nil {
		result.Err = errs.Error()
	}
	if find != nil {
		result.Subject = find.Subject
		result.Body = find.Body
	} else if result.Err == "" {
		result.Err = "searched mail not found"
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = fmt.Sprintf("%s", elapsed)
	result.Executor.IMAPPassword = "****hidden****" // do not output password

	return executors.Dump(result)
}

func (e *Executor) getMail(l venom.Logger) (*Mail, error) {
	if e.SearchFrom == "" && e.SearchSubject == "" && e.SearchBody == "" {
		return nil, fmt.Errorf("You have to use searchfrom and/or searchsubject and/or searchbody")
	}

	c, errc := connect(e.IMAPHost, e.IMAPPort, e.IMAPUser, e.IMAPPassword)
	if errc != nil {
		return nil, fmt.Errorf("Error while connecting:%s", errc.Error())
	}
	defer c.Logout(5 * time.Second)

	var box string

	if e.MBox == "" {
		box = "INBOX"
	} else {
		box = e.MBox
	}

	count, err := queryCount(c, box)
	if err != nil {
		return nil, fmt.Errorf("Error while queryCount:%s", err.Error())
	}

	l.Debugf("count messages:%d", count)

	if count == 0 {
		return nil, errors.New("No message to fetch")
	}

	messages, err := fetch(c, box, count, l)
	if err != nil {
		return nil, fmt.Errorf("Error while feching messages:%s", err.Error())
	}

	for _, msg := range messages {
		m, erre := extract(msg, l)
		if erre != nil {
			return nil, erre
		}

		found, errs := e.isSearched(m)
		if errs != nil {
			return nil, errs
		}

		if found {
			if e.DeleteOnSuccess {
				l.Debugf("Delete message %s", m.UID)
				if err := m.delete(c); err != nil {
					return nil, err
				}
			} else if e.MBoxOnSuccess != "" {
				l.Debugf("Move to %s", e.MBoxOnSuccess)
				if err := m.move(c, e.MBoxOnSuccess); err != nil {
					return nil, err
				}
			}
			return m, nil
		}
	}

	return nil, errors.New("Mail not found")
}

func (e *Executor) isSearched(m *Mail) (bool, error) {
	if e.SearchFrom != "" {
		ma, erra := regexp.MatchString(e.SearchFrom, m.From)
		if erra != nil || !ma {
			return false, erra
		}
	}
	if e.SearchSubject != "" {
		mb, errb := regexp.MatchString(e.SearchSubject, m.Subject)
		if errb != nil || !mb {
			return false, errb
		}
	}
	if e.SearchBody != "" {
		mc, errc := regexp.MatchString(e.SearchBody, m.Body)
		if errc != nil || !mc {
			return false, errc
		}
	}
	return true, nil
}

func (m *Mail) move(c *imap.Client, mbox string) error {
	seq, _ := imap.NewSeqSet("")
	seq.AddNum(m.UID)

	if _, err := c.UIDMove(seq, mbox); err != nil {
		return fmt.Errorf("Error while move msg to %s, err:%s", mbox, err.Error())
	}
	return nil
}

func (m *Mail) delete(c *imap.Client) error {
	seq, _ := imap.NewSeqSet("")
	seq.AddNum(m.UID)

	if _, err := c.UIDStore(seq, "+FLAGS.SILENT", imap.NewFlagSet(`\Deleted`)); err != nil {
		return fmt.Errorf("Error while deleting msg, err: %s", err.Error())
	}
	if _, err := c.Expunge(nil); err != nil {
		return fmt.Errorf("Error while expunging messages: err: %s", err.Error())
	}
	return nil
}

func connect(host, port, imapUsername, imapPassword string) (*imap.Client, error) {
	if !strings.Contains(host, ":") {
		if port == "" {
			port = ":993"
		} else if port != "" && !strings.HasPrefix(port, ":") {
			port = ":" + port
		}
	}

	c, errd := imap.DialTLS(host+port, nil)
	if errd != nil {
		return nil, fmt.Errorf("Unable to dial: %s", errd)
	}

	if c.Caps["STARTTLS"] {
		if _, err := check(c.StartTLS(nil)); err != nil {
			return nil, fmt.Errorf("Unable to start TLS: %s\n", err)
		}
	}

	c.SetLogMask(imapSafeLogMask)
	if _, err := check(c.Login(imapUsername, imapPassword)); err != nil {
		return nil, fmt.Errorf("Unable to login: %s", err)
	}
	c.SetLogMask(imapLogMask)

	return c, nil
}

func fetch(c *imap.Client, box string, nb uint32, l venom.Logger) ([]imap.Response, error) {
	l.Debugf("call Select")
	if _, err := c.Select(box, false); err != nil {
		l.Errorf("Error with select %s", err.Error())
		return []imap.Response{}, err
	}
	defer c.Close(false)

	seqset, _ := imap.NewSeqSet("1:*")

	cmd, err := c.Fetch(seqset, "ENVELOPE", "RFC822.HEADER", "RFC822.TEXT", "UID")
	if err != nil {
		l.Errorf("Error with fetch:%s", err)
		return []imap.Response{}, err
	}

	messages := []imap.Response{}
	for cmd.InProgress() {
		// Wait for the next response (no timeout)
		c.Recv(-1)

		// Process command data
		for _, rsp := range cmd.Data {
			messages = append(messages, *rsp)
		}
		cmd.Data = nil
		c.Data = nil
	}
	l.Debugf("Nb messages fetch:%d", len(messages))
	return messages, nil
}

func queryCount(imapClient *imap.Client, box string) (uint32, error) {
	cmd, errc := check(imapClient.Status(box))
	if errc != nil {
		return 0, errc
	}

	var count uint32
	for _, result := range cmd.Data {
		mailboxStatus := result.MailboxStatus()
		if mailboxStatus != nil {
			count += mailboxStatus.Messages
		}
	}

	return count, nil
}

func check(cmd *imap.Command, erri error) (*imap.Command, error) {
	if erri != nil {
		return nil, erri
	}

	if _, err := cmd.Result(imap.OK); err != nil {
		return nil, err
	}

	return cmd, nil
}
