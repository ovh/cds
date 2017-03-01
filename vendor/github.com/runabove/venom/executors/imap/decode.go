package imap

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"

	log "github.com/Sirupsen/logrus"
	"github.com/yesnault/go-imap/imap"
)

func decodeHeader(msg *mail.Message, headerName string) (string, error) {
	dec := new(mime.WordDecoder)
	s, err := dec.DecodeHeader(msg.Header.Get(headerName))
	if err != nil {
		return msg.Header.Get(headerName), fmt.Errorf("Error while decode header %s:%s", headerName, msg.Header.Get(headerName))
	}
	return s, nil
}

func hash(in string) string {
	h2 := md5.New()
	io.WriteString(h2, in)
	return fmt.Sprintf("%x", h2.Sum(nil))
}

func extract(rsp imap.Response, l *log.Entry) (*Mail, error) {
	tm := &Mail{}
	var params map[string]string

	header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
	tm.UID = imap.AsNumber((rsp.MessageInfo().Attrs["UID"]))
	body := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.TEXT"])
	if mmsg, _ := mail.ReadMessage(bytes.NewReader(header)); mmsg != nil {
		var errds error
		tm.Subject, errds = decodeHeader(mmsg, "Subject")
		if errds != nil {
			return nil, errds
		}
		var errdf error
		tm.From, errdf = decodeHeader(mmsg, "From")
		if errdf != nil {
			return nil, fmt.Errorf("Error while read From field:%s", errdf)
		}

		var errpm error
		_, params, errpm = mime.ParseMediaType(mmsg.Header.Get("Content-Type"))
		if errpm != nil {
			return nil, fmt.Errorf("Error while read Content-Type:%s", errpm)
		}
	}

	r := quotedprintable.NewReader(bytes.NewReader(body))
	bodya, errra := ioutil.ReadAll(r)
	if errra == nil {
		tm.Body = string(bodya)
		return tm, nil
	} else if len(params) > 0 {
		r := bytes.NewReader(body)
		mr := multipart.NewReader(r, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				continue
			}
			if err != nil {
				l.Debugf("Error while read Part:%s", err)
				break
			}
			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				l.Debugf("Error while ReadAll Part:%s", err)
				continue
			}
			tm.Body = string(slurp)
			break
		}
	}

	if tm.Body == "" {
		tm.Body = string(bodya)
	}
	return tm, nil
}
