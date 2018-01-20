package user

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/mail"
	"net/smtp"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

const templCreate = `Welcome to Tat,

You recently signed up for Tat.

{{.TextVerify}}
{{.CMDVerify}}

Regards,
--
Tat Team
`

const templReset = `Hello,

You recently asked to reset your Tat password. Ignore this email if not.

{{.TextVerify}}
{{.CMDVerify}}

Regards,
--
Tat Team
`

type paramEmail struct {
	TextVerify string
	CMDVerify  string
}

//SendVerifyEmail send a mail to new user, ask him to valide his new account
func SendVerifyEmail(username, to, tokenVerify, device string) error {
	text, cmd := getSigninCmd(username, tokenVerify, device)
	return sendEmail(templCreate, "Welcome to Tat", username, to, tokenVerify, text, cmd, device)
}

//SendAskResetEmail send a mail to user, ask him to confirm reset password
func SendAskResetEmail(username, to, tokenVerify, device string) error {
	text, cmd := getResetCmd(username, tokenVerify, device)
	return sendEmail(templReset, "Tat : confirm your reset password", username, to, tokenVerify, text, cmd, device)
}

func sendEmail(templ, subject, username, toUser, tokenVerify, text, cmd, device string) error {
	t := template.New("Email template")
	t, err := t.Parse(templ)
	if err != nil {
		log.Errorf("Error with parsing template:%s ", err.Error())
		return err
	}

	paramEmail := &paramEmail{
		TextVerify: text,
		CMDVerify:  cmd,
	}

	var b bytes.Buffer
	err = t.Execute(&b, paramEmail)
	if err != nil {
		log.Errorf("Error with Execute template:%s ", err.Error())
		return err
	}

	if viper.GetBool("no_smtp") {
		fmt.Println("##### NO SMTP DISPLAY MAIL IN CONSOLE ######")
		fmt.Printf("Subject:%s\n", subject)
		fmt.Printf("Text:%s\n", b.Bytes())
		fmt.Println("##### END MAIL ######")
		return nil
	}

	from := mail.Address{
		Name:    "",
		Address: viper.GetString("smtp_from"),
	}
	to := mail.Address{
		Name:    "",
		Address: toUser,
	}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = viper.GetString("smtp_from")
	headers["To"] = to.String()
	headers["Subject"] = subject

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + b.String()

	// Connect to the SMTP Server
	servername := fmt.Sprintf("%s:%s", viper.GetString("smtp_host"), viper.GetString("smtp_port"))

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         viper.GetString("smtp_host"),
	}

	var c *smtp.Client
	if viper.GetBool("smtp_tls") {

		// Here is the key, you need to call tls.Dial instead of smtp.Dial
		// for smtp servers running on 465 that require an ssl connection
		// from the very beginning (no starttls)

		conn, errc := tls.Dial("tcp", servername, tlsconfig)
		if errc != nil {
			log.Errorf("Error with c.Dial:%s", errc.Error())
			return err
		}

		c, err = smtp.NewClient(conn, viper.GetString("smtp_host"))
		if err != nil {
			log.Errorf("Error with c.NewClient:%s", err.Error())
			return err
		}
	} else {
		c, err = smtp.Dial(servername)
		if err != nil {
			log.Errorf("Error while smtp.Dial:%s", err)
		}
		defer c.Close()
	}

	// Auth
	if viper.GetString("smtp_user") != "" && viper.GetString("smtp_password") != "" {
		auth := smtp.PlainAuth("", viper.GetString("smtp_user"), viper.GetString("smtp_password"), viper.GetString("smtp_host"))
		if err = c.Auth(auth); err != nil {
			log.Errorf("Error with c.Auth:%s", err.Error())
			return err
		}
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		log.Errorf("Error with c.Mail:%s", err.Error())
		return err
	}

	if err = c.Rcpt(to.Address); err != nil {
		log.Errorf("Error with c.Rcpt:%s", err.Error())
		return err
	}

	// Data
	w, err := c.Data()
	if err != nil {
		log.Errorf("Error with c.Data:%s", err.Error())
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Errorf("Error with c.Write:%s", err.Error())
		return err
	}

	err = w.Close()
	if err != nil {
		log.Errorf("Error with c.Close:%s", err.Error())
		return err
	}

	c.Quit()

	return nil
}
