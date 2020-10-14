package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"sync/atomic"
	"text/template"

	"github.com/jordan-wright/email"
	"github.com/ovh/cds/sdk"
)

var smtpUser, smtpPassword, smtpFrom, smtpHost, smtpPort, smtpModeTLS string
var smtpTLS, smtpEnable, smtpInsecureSkipVerify bool
var lastError error
var counter uint64

const (
	// modeTLS uses tls without starttls
	modeTLS = "tls"
	// modeStartTLS uses starttls
	modeStartTLS = "starttls"
)

const templateSignedup = `Welcome to CDS,

You recently signed up for CDS.

To verify your email address, follow this link:
{{.URL}}

If you are using the command line, you can run:

$ cdsctl signup verify --api-url {{.APIURL}} {{.Token}}

Regards,
--
CDS Team
`

const templateAskReset = `Hi {{.Username}},

You asked for a password reset.

Follow this link to set a new password on your account:
{{.URL}}


If you are using the command line, you can run:

$ cdsctl reset-password confirm --api-url {{.APIURL}} {{.Token}}

Regards,
--
CDS Team
`

// Init initializes configuration
func Init(user, password, from, host, port, modeTLS string, insecureSkipVerify, disable bool) {
	smtpUser = user
	smtpPassword = password
	smtpFrom = from
	smtpHost = host
	smtpPort = port
	smtpModeTLS = modeTLS
	smtpInsecureSkipVerify = insecureSkipVerify
	smtpEnable = !disable
}

// Status verification of smtp configuration, returns OK or KO
func Status(ctx context.Context) sdk.MonitoringStatusLine {
	if !smtpEnable {
		return sdk.MonitoringStatusLine{Component: "SMTP", Value: "Conf: SMTP Disabled", Status: sdk.MonitoringStatusWarn}
	}
	if lastError != nil {
		return sdk.MonitoringStatusLine{Component: "SMTP", Value: "KO: " + lastError.Error(), Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{Component: "SMTP", Value: fmt.Sprintf("OK (%d sent)", counter), Status: sdk.MonitoringStatusOK}
}

// SendMailVerifyToken send mail to verify user account.
func SendMailVerifyToken(ctx context.Context, userMail, username, token, callbackUI, APIURL string) error {
	callbackURL := fmt.Sprintf(callbackUI, token)

	mailContent, err := createTemplate(templateSignedup, callbackURL, APIURL, username, token)
	if err != nil {
		return err
	}

	return SendEmail(ctx, "[CDS] Welcome to CDS", &mailContent, userMail, false)
}

// SendMailAskResetToken send mail to ask reset a user account.
func SendMailAskResetToken(ctx context.Context, userMail, username, token, callbackUI, APIURL string) error {
	callbackURL := fmt.Sprintf(callbackUI, token)

	mailContent, err := createTemplate(templateAskReset, callbackURL, APIURL, username, token)
	if err != nil {
		return err
	}

	return SendEmail(ctx, "[CDS] Reset your password", &mailContent, userMail, false)
}

func createTemplate(templ, callbackURL, callbackAPIURL, username, token string) (bytes.Buffer, error) {
	var b bytes.Buffer

	// Create mail template
	t := template.New("Email template")
	t, err := t.Parse(templ)
	if err != nil {
		return b, sdk.WrapError(err, "error with parsing template")
	}

	if err := t.Execute(&b, struct{ URL, APIURL, Username, Token string }{callbackURL, callbackAPIURL, username, token}); err != nil {
		return b, sdk.WrapError(err, "cannot execute template")
	}

	return b, nil
}

//SendEmail is the core function to send an email
func SendEmail(ctx context.Context, subject string, mailContent *bytes.Buffer, userMail string, isHTML bool) error {
	e := email.NewEmail()
	e.From = smtpFrom
	e.To = []string{userMail}
	e.Subject = subject
	e.Text = mailContent.Bytes()
	if isHTML {
		e.HTML = mailContent.Bytes()
	}

	if !smtpEnable {
		fmt.Println("##### NO SMTP DISPLAY MAIL IN CONSOLE ######")
		fmt.Printf("Subject:%s\n", subject)
		fmt.Printf("Text:%s\n", string(e.Text))
		fmt.Println("##### END MAIL ######")
		return nil
	}
	servername := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	var auth smtp.Auth
	if smtpUser != "" && smtpPassword != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	}

	tlsconfig := &tls.Config{
		InsecureSkipVerify: smtpInsecureSkipVerify,
		ServerName:         smtpHost,
	}

	var err error
	switch smtpModeTLS {
	case modeStartTLS:
		err = e.SendWithStartTLS(servername, auth, tlsconfig)
	case modeTLS:
		err = e.SendWithTLS(servername, auth, tlsconfig)
	default:
		err = e.Send(servername, auth)
	}
	if err != nil {
		lastError = err
	} else {
		atomic.AddUint64(&counter, 1)
		lastError = nil
	}
	return sdk.WithStack(err)
}
