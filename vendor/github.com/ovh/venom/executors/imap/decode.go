package imap

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"

	log "github.com/sirupsen/logrus"
	"github.com/yesnault/go-imap/imap"

	"github.com/ovh/venom"
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

func extract(rsp imap.Response, l venom.Logger) (*Mail, error) {
	tm := &Mail{}

	header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
	tm.UID = imap.AsNumber((rsp.MessageInfo().Attrs["UID"]))
	body := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.TEXT"])

	mmsg, err := mail.ReadMessage(bytes.NewReader(header))
	if err != nil {
		return nil, err
	}
	tm.Subject, err = decodeHeader(mmsg, "Subject")
	if err != nil {
		log.Warnf("Cannot decode Subject header: %s", err)
		return nil, nil
	}
	tm.From, err = decodeHeader(mmsg, "From")
	if err != nil {
		log.Warnf("Cannot decode From header: %s", err)
		return nil, nil
	}
	tm.To, err = decodeHeader(mmsg, "To")
	if err != nil {
		log.Warnf("Cannot decode To header: %s", err)
		return nil, nil
	}

	encoding := mmsg.Header.Get("Content-Transfer-Encoding")
	var r io.Reader = bytes.NewReader(body)
	switch encoding {
	case "7bit", "8bit", "binary":
		// noop, reader already initialized.
	case "quoted-printable":
		r = quotedprintable.NewReader(r)
	case "base64":
		r = base64.NewDecoder(base64.StdEncoding, r)
	}
	l.Debugf("Mail Content-Transfer-Encoding is %s ", encoding)

	contentType, params, err := mime.ParseMediaType(mmsg.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("Error while reading Content-Type:%s", err)
	}
	if contentType == "multipart/mixed" || contentType == "multipart/alternative" {
		if boundary, ok := params["boundary"]; ok {
			mr := multipart.NewReader(r, boundary)
			for {
				p, errm := mr.NextPart()
				if errm == io.EOF {
					continue
				}
				if errm != nil {
					l.Debugf("Error while read Part:%s", err)
					break
				}
				slurp, errm := ioutil.ReadAll(p)
				if errm != nil {
					l.Debugf("Error while ReadAll Part:%s", err)
					continue
				}
				tm.Body = string(slurp)
				break
			}
		}
	} else {
		body, err = ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
	}
	if tm.Body == "" {
		tm.Body = string(body)
	}
	return tm, nil
}
