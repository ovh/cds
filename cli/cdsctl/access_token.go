package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

func consumer() *cobra.Command {
	var (
		cmd = cli.Command{
			Name:  "xconsumer",
			Short: "Manage CDS auth consumers [EXPERIMENTAL]",
		}

		listbyUserCmd = cli.Command{
			Name:  "list",
			Short: "List your auth consumers",
			Flags: []cli.Flag{
				{
					Name:      "group",
					Type:      cli.FlagSlice,
					ShortHand: "g",
					Usage:     "filter by group",
				},
			},
		}

		newCmd = cli.Command{
			Name:  "new",
			Short: "Create a new access token",
			Flags: []cli.Flag{
				{
					Name:      "description",
					ShortHand: "d",
					Usage:     "what is the purpose of this token",
				}, {
					Name:      "expiration",
					ShortHand: "e",
					Usage:     "expiration delay of the token (1d, 24h, 1440m, 86400s)",
					Default:   "1d",
					IsValid: func(s string) bool {
						return true
					},
				}, {
					Name:      "group",
					Type:      cli.FlagSlice,
					ShortHand: "g",
					Usage:     "define the scope of the token through groups",
				},
			},
		}

		/*regenCmd = cli.Command{
			Name:  "regen",
			Short: "Regenerate access token",
			VariadicArgs: cli.Arg{
				Name:       "token-id",
				AllowEmpty: false,
			},
		}

		deleteCmd = cli.Command{
			Name:  "delete",
			Short: "Delete access token",
			VariadicArgs: cli.Arg{
				Name:       "token-id",
				AllowEmpty: true,
			},
		}*/
	)

	return cli.NewCommand(cmd, nil,
		cli.SubCommands{
			cli.NewListCommand(listbyUserCmd, accesstokenListRun, nil),
			cli.NewCommand(newCmd, accesstokenNewRun, nil),
			//cli.NewCommand(regenCmd, accesstokenRegenRun, nil),
			//cli.NewCommand(deleteCmd, accesstokenDeleteRun, nil),
		},
	)
}

func accesstokenListRun(v cli.Values) (cli.ListResult, error) {
	/*type displayToken struct {
		ID          string `cli:"id,key"`
		Description string `cli:"description"`
		UserName    string `cli:"user"`
		ExpireAt    string `cli:"expired_at"`
		Created     string `cli:"created"`
		Status      string `cli:"status"`
		Scope       string `cli:"scope"`
	}

	var displayTokenFunc = func(t sdk.AccessToken) displayToken {
		var groupNames []string
		for _, g := range t.Groups {
			groupNames = append(groupNames, g.Name)
		}
		return displayToken{
			ID: t.ID,
			// TODO
			//Description: t.Description,
			//UserName:    t.User.Fullname,
			ExpireAt: t.ExpireAt.Format(time.RFC850),
			Created:  t.Created.Format(time.RFC850),
			Status:   t.Status,
			Scope:    strings.Join(groupNames, ","),
		}
	}

	var displayAllTokensFunc = func(ts []sdk.AccessToken) []displayToken {
		var res = make([]displayToken, len(ts))
		for i := range ts {
			res[i] = displayTokenFunc(ts[i])
		}
		return res
	}

	groups := v.GetStringSlice("group")
	if len(groups) == 0 {
		tokens, err := client.AccessTokenListByUser(cfg.User)
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(displayAllTokensFunc(tokens)), nil
	}

	tokens, err := client.AccessTokenListByGroup(groups...)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(displayAllTokensFunc(tokens)), nil*/

	return nil, nil
}

func accesstokenNewRun(v cli.Values) error {
	/*allGroups, err := client.GroupList()
	if err != nil {
		return err
	}

	description := v.GetString("description")
	expiration := v.GetString("expiration")
	groups := v.GetStringSlice("group")

	// If the flag has not been set, ask interactively
	if description == "" {
		description = cli.AskValueChoice("Description")
	}
	if expiration == "" {
		expiration = cli.AskValueChoice("Expiration")
	}
	if len(groups) == 0 {
		var groupNames []string
		for _, g := range allGroups {
			groupNames = append(groupNames, g.Name)
		}
		choices := cli.MultiSelect("Groups", groupNames...)
		for _, choice := range choices {
			groups = append(groups, groupNames[choice])
		}
	}

	// Compute expiration string
	var r = regexp.MustCompile("([0-9])(s|m|h|d)")
	if !r.MatchString(expiration) {
		return errors.New("unsupported expiration expression")
	}

	matches := r.FindStringSubmatch(expiration)
	factor, _ := strconv.ParseFloat(matches[1], 64)
	unit := time.Second
	switch matches[2] {
	case "m":
		unit = time.Minute
	case "h":
		unit = time.Hour
	case "d":
		unit = 24 * time.Hour
	}

	expirationDuration := time.Duration(factor) * unit

	// Retrieve group IDs from all the groups accessible by the user
	var groupsIDs []int64
	for _, group := range groups {
		var groupFound bool
		for _, knowGroup := range allGroups {
			if knowGroup.Name == group {
				groupFound = true
				groupsIDs = append(groupsIDs, knowGroup.ID)
				break
			}
		}
		if !groupFound {
			return errors.New("group not found")
		}
	}

	var request = sdk.AccessTokenRequest{
		Description:           description,
		ExpirationDelaySecond: expirationDuration.Seconds(),
		GroupsIDs:             groupsIDs,
		Origin:                "cdsctl",
	}

	t, jwt, err := client.AccessTokenCreate(request)
	if err != nil {
		return fmt.Errorf("unable to create access token: %v", err)
	}
	fmt.Println()

	displayToken(t, jwt)
	*/
	return nil
}

/*func displayToken(t sdk.AccessToken, jwt string) {
	fmt.Println("Token successfully generated")
	fmt.Println(cli.Cyan("ID"), "\t\t", t.ID)
	// TODO
	//fmt.Println(cli.Cyan("Description"), "\t", t.Description)
	fmt.Println(cli.Cyan("Creation"), "\t", t.Created.Format(time.RFC850))
	fmt.Println(cli.Red("Expiration"), "\t", cli.Red(t.ExpireAt.Format(time.RFC850)))
	// TODO
	//fmt.Println(cli.Cyan("User"), "\t\t", t.User.Fullname)
	var groupNames []string
	for _, g := range t.Groups {
		groupNames = append(groupNames, g.Name)
	}
	fmt.Println(cli.Cyan("Scope"), "\t\t", groupNames)
	fmt.Println()
	fmt.Println(cli.Red("Here it is, keep it in a safe place, it will never ne displayed again."))
	fmt.Println(jwt)
}*/

/*
func accesstokenRegenRun(v cli.Values) error {
	tokenIDs := v.GetStringSlice("token-id")
	for _, id := range tokenIDs {

		t, jwt, err := client.AccessTokenRegen(id)
		if err != nil {
			fmt.Println("unable to regen token", id, cli.Red(err.Error()))
		}

		displayToken(t, jwt)
	}

	return nil
}

func accesstokenDeleteRun(v cli.Values) error {
	tokenIDs := v.GetStringSlice("token-id")
	for _, id := range tokenIDs {
		if err := client.AccessTokenDelete(id); err != nil {
			fmt.Println("unable to delete token", id, cli.Red(err.Error()))
		}

	}

	return nil
}
*/
