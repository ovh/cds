package user

import (
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(cmdUserList)
	Cmd.AddCommand(cmdUserMe)
	Cmd.AddCommand(cmdUserContacts)
	Cmd.AddCommand(cmdUserAddContact)
	Cmd.AddCommand(cmdUserRemoveContact)
	Cmd.AddCommand(cmdUserAddFavoriteTopic)
	Cmd.AddCommand(cmdUserRemoveFavoriteTopic)
	Cmd.AddCommand(cmdUserEnableNotificationsTopic)
	Cmd.AddCommand(cmdUserEnableNotificationsAllTopics)
	Cmd.AddCommand(cmdUserDisableNotificationsTopic)
	Cmd.AddCommand(cmdUserDisableNotificationsAllTopics)
	Cmd.AddCommand(cmdUserAddFavoriteTag)
	Cmd.AddCommand(cmdUserRemoveFavoriteTag)
	Cmd.AddCommand(cmdUserAdd)
	Cmd.AddCommand(cmdUserReset)
	Cmd.AddCommand(cmdUserResetSystem)
	Cmd.AddCommand(cmdUserConvertToSystem)
	Cmd.AddCommand(cmdUserUpdateSystem)
	Cmd.AddCommand(cmdUserArchive)
	Cmd.AddCommand(cmdUserRename)
	Cmd.AddCommand(cmdUserUpdate)
	Cmd.AddCommand(cmdUserSetAdmin)
	Cmd.AddCommand(cmdUserVerify)
	Cmd.AddCommand(cmdUserCheck)
}

// Cmd user
var Cmd = &cobra.Command{
	Use:     "user",
	Short:   "User commands: tatcli user --help",
	Long:    `User commands: tatcli user <command>`,
	Aliases: []string{"u"},
}
