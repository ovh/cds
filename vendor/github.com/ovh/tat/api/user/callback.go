package user

import (
	"fmt"

	"strings"

	"github.com/spf13/viper"
)

// GetSigninCmd returns the command to exute to finish validate Account
func getSigninCmd(username, tokenVerify, callback string) (string, string) {

	textVerify := "To verify your email address, follow this link : "

	// tatcli  --url=.... user verify --save
	if strings.HasPrefix(callback, "user verify --save") { // tatcli
		textVerify = "To verify your email address, execute this command : "
	}

	return textVerify, getVerifyCmd(username, tokenVerify, callback)
}

// GetResetCmd returns the command to exute to finish validate Reset
func getResetCmd(username, tokenVerify, callback string) (string, string) {

	textVerify := "To complete your password resetting, follow this link:"

	// tatcli  --url=.... user verify --save
	if strings.HasPrefix(callback, "user verify --save") {
		textVerify = "To complete your password resetting, execute this command : "
	}

	return textVerify, getVerifyCmd(username, tokenVerify, callback)
}

func getVerifyCmd(username, tokenVerify, callback string) string {

	if callback == "" {
		return fmt.Sprintf("%s://%s:%s%s/user/verify/%s/%s",
			viper.GetString("exposed_scheme"), viper.GetString("exposed_host"), viper.GetString("exposed_port"), viper.GetString("exposed_path"), username, tokenVerify)
	}
	c := strings.Replace(callback, ":scheme", viper.GetString("exposed_scheme"), -1)
	c = strings.Replace(c, ":host", viper.GetString("exposed_host"), -1)
	c = strings.Replace(c, ":port", viper.GetString("exposed_port"), -1)
	c = strings.Replace(c, ":path", viper.GetString("exposed_path"), -1)
	c = strings.Replace(c, ":username", username, -1)
	c = strings.Replace(c, ":token", tokenVerify, -1)

	if strings.HasPrefix(username, "tat.system") {
		fmt.Printf("Url CallBack generated : %s\n", c)
	}

	return c
}
