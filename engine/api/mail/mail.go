package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"
	"text/template"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var smtpUser, smtpPassword, smtpFrom, smtpHost, smtpPort string
var smtpTLS, smtpEnable bool

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

const templateReset = `Hi {{.Username}},

Your password was successfully reset.

Regards,
--
CDS Team
`

// Init initializes configuration
func Init(user, password, from, host, port string, tls, disable bool) {
	smtpUser = user
	smtpPassword = password
	smtpFrom = from
	smtpHost = host
	smtpPort = port
	smtpTLS = tls
	smtpEnable = !disable
}

// Status verification of smtp configuration, returns OK or KO
func Status(ctx context.Context) sdk.MonitoringStatusLine {
	if _, err := smtpClient(ctx); err != nil {
		return sdk.MonitoringStatusLine{Component: "SMTP Ping", Value: "KO: " + err.Error(), Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{Component: "SMTP Ping", Value: "Connect OK", Status: sdk.MonitoringStatusOK}
}

func smtpClient(ctx context.Context) (*smtp.Client, error) {
	if smtpHost == "" || smtpPort == "" || !smtpEnable {
		return nil, errors.New("No SMTP configuration")
	}

	// Connect to the SMTP Server
	servername := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpHost,
	}

	var c *smtp.Client
	var err error
	if smtpTLS {
		// Here is the key, you need to call tls.Dial instead of smtp.Dial
		// for smtp servers running on 465 that require an ssl connection
		// from the very beginning (no starttls)
		conn, errc := tls.Dial("tcp", servername, tlsconfig)
		if errc != nil {
			log.Warning(ctx, "Error with c.Dial:%s\n", errc.Error())
			return nil, errc
		}

		c, err = smtp.NewClient(conn, smtpHost)
		if err != nil {
			log.Warning(ctx, "Error with c.NewClient:%s\n", err.Error())
			return nil, err
		}
		// TLS config
		tlsconfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         smtpHost,
		}
		c.StartTLS(tlsconfig)
	} else {
		c, err = smtp.Dial(servername)
		if err != nil {
			log.Warning(ctx, "Error with c.NewClient:%s\n", err.Error())
			return nil, err
		}
	}

	// Auth
	if smtpUser != "" && smtpPassword != "" {
		auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
		if err = c.Auth(auth); err != nil {
			log.Warning(ctx, "Error with c.Auth:%s\n", err.Error())
			c.Close()
			return nil, err
		}
	}
	return c, nil
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

// SendMailResetToken send mail to reset a user account.
func SendMailResetToken(ctx context.Context, userMail, username, token, callback string) error {
	callbackURL := fmt.Sprintf(callback, token)

	mailContent, err := createTemplate(templateReset, callbackURL, "", username, "")
	if err != nil {
		return err
	}

	return SendEmail(ctx, "[CDS] Your password was reset", &mailContent, userMail, false)
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
	from := mail.Address{
		Name:    "",
		Address: smtpFrom,
	}
	to := mail.Address{
		Name:    "",
		Address: userMail,
	}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = smtpFrom
	headers["To"] = to.String()
	headers["Subject"] = subject

	if isHTML {
		headers["Content-Type"] = `text/html; charset="utf-8"`
	}

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + mailContent.String()

	if !smtpEnable {
		fmt.Println("##### NO SMTP DISPLAY MAIL IN CONSOLE ######")
		fmt.Printf("Subject:%s\n", subject)
		fmt.Printf("Text:%s\n", message)
		fmt.Println("##### END MAIL ######")
		return nil
	}

	c, err := smtpClient(ctx)
	if err != nil {
		return sdk.WrapError(err, "Cannot get smtp client")
	}
	defer c.Close()

	// To && From
	if err = c.Mail(from.Address); err != nil {
		return sdk.WrapError(err, "Error with c.Mail")
	}

	if err = c.Rcpt(to.Address); err != nil {
		return sdk.WrapError(err, "Error with c.Rcpt")
	}

	// Data
	w, err := c.Data()
	if err != nil {
		return sdk.WrapError(err, "Error with c.Data")
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return sdk.WrapError(err, "Error with c.Write")
	}

	err = w.Close()
	if err != nil {
		return sdk.WrapError(err, "Error with c.Close")
	}

	c.Quit()

	return nil
}
