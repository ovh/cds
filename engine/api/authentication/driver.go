package authentication

// LOCAL
// signup
// not auth
//   check username don't exists for authentified user
//   create new consumer for authentified user with password encrypted
//   generate new session for consumer
//   generate jwt and returns it in cookie.
// auth
//   create no local consumer for authentified user
//   create new local consumer with given password
//   generate new session for consumer
//   generate jwt and returns it in cookie.

// signin
//   check username exists for authentified user
//   check if a consumer local exists, if true compare password
//   generate new session for consumer
//   ...

// reset password
//   check username exists for authentified user and consumer local exists
//   send a mail to the principal confimed email
//   handle form with new password, check regen token
//   store the new password in consumer

// GITHUB
// not auth
//   callback from github
//   get github user with token
//   check if a consumer exists with github user id
//   if yes return a new session with jwt	for consumer
//   if no create create new user and consumer
// auth
//   callback from github
//   get github user with token
//   check that no consumer if type github exists for current user
//   if no return create a new consumer and return a new session with jwt	for consumer

// SSO
// not auth
//   callback from sso
//   extract info from sso token, add it to cookie
//   check if a consumer exists with username sso
//   if no create a new user and customer and return a new session with jwt
//   if yes return a new session with jwt	for consumer
// auth
//   callback from sso
//   check that no consumer exists for sso
//   if no create a new consumer and return session jwt
