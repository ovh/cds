package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"
	"text/template"

	"github.com/ovh/cds/sdk/log"
)

type emailParam struct {
	URL string
}

var smtpUser, smtpPassword, smtpFrom, smtpHost, smtpPort string
var smtpTLS, smtpEnable bool

const templateSignedUP = `Welcome to CDS,

You recently signed up for CDS.

To verify your email address, follow this link :
{{.URL}}

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
func Status() string {
	if _, err := smtpClient(); err != nil {
		return fmt.Sprintf("KO (%s)", err)
	}
	return "OK"
}

func smtpClient() (*smtp.Client, error) {
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
			log.Warning("Error with c.Dial:%s\n", errc.Error())
			return nil, errc
		}

		c, err = smtp.NewClient(conn, smtpHost)
		if err != nil {
			log.Warning("Error with c.NewClient:%s\n", err.Error())
			return nil, err
		}
	} else {
		c, err = smtp.Dial(servername)
		if err != nil {
			log.Warning("Error with c.NewClient:%s\n", err.Error())
			return nil, err
		}
	}

	// Auth
	if smtpUser != "" && smtpPassword != "" {
		auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
		if err = c.Auth(auth); err != nil {
			log.Warning("Error with c.Auth:%s\n", err.Error())
			c.Close()
			return nil, err
		}
	}
	return c, nil
}

// SendMailVerifyToken Send mail to verify user account
func SendMailVerifyToken(userMail, username, token, callback string) error {
	callbackURL := getCallbackURL(username, token, callback)

	mailContent, err := createTemplate(templateSignedUP, callbackURL)
	if err != nil {
		return err
	}
	subject := "Welcome to CDS"
	if !smtpEnable {
		fmt.Println("##### NO SMTP DISPLAY MAIL IN CONSOLE ######")
		fmt.Printf("Subject:%s\n", subject)
		fmt.Printf("Text:%s\n", mailContent.Bytes())
		fmt.Println("##### END MAIL ######")
		return nil
	}
	return SendEmail(subject, &mailContent, userMail)
}

func getCallbackURL(username, token, callback string) string {
	if callback == "cdscli" {
		return fmt.Sprintf("cds user verify %s %s", username, token)
	}
	return fmt.Sprintf(callback, username, token)
}

func createTemplate(templ, callbackURL string) (bytes.Buffer, error) {
	var b bytes.Buffer

	// Create mail template
	t := template.New("Email template")
	t, err := t.Parse(templ)
	if err != nil {
		fmt.Printf("Error with parsing template:%s \n", err.Error())
		return b, err
	}

	param := emailParam{
		URL: callbackURL,
	}
	err = t.Execute(&b, param)
	if err != nil {
		fmt.Printf("Error with Execute template:%s \n", err.Error())
		return b, err
	}

	return b, nil
}

//SendEmail is the core function to send an email
func SendEmail(subject string, mailContent *bytes.Buffer, userMail string) error {

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

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + mailContent.String()

	c, err := smtpClient()
	if err != nil {
		log.Warning("Cannot get smtp client:%s\n", err.Error())
		return err
	}
	defer c.Close()

	// To && From
	if err = c.Mail(from.Address); err != nil {
		log.Warning("Error with c.Mail:%s\n", err.Error())
		return err
	}

	if err = c.Rcpt(to.Address); err != nil {
		log.Warning("Error with c.Rcpt:%s\n", err.Error())
		return err
	}

	// Data
	w, err := c.Data()
	if err != nil {
		log.Warning("Error with c.Data:%s\n", err.Error())
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Warning("Error with c.Write:%s", err.Error())
		return err
	}

	err = w.Close()
	if err != nil {
		log.Warning("Error with c.Close:%s", err.Error())
		return err
	}

	c.Quit()

	return nil
}
