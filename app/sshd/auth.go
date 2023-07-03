package sshd

import (
	"mjump/app/logger"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// type AuthStatus ssh.AuthResult

// const (
// 	AuthFailed              = AuthStatus(ssh.AuthFailed)
// 	AuthSuccessful          = AuthStatus(ssh.AuthSuccessful)
// 	AuthPartiallySuccessful = AuthStatus(ssh.AuthPartiallySuccessful)
// )

func SSHKeyboardInteractiveAuth(ctx ssh.Context, challenger gossh.KeyboardInteractiveChallenge) (res bool) {
	logger.Debug("用户密码认证")
	return true
	// return AuthFailed
	// if value, ok := ctx.Value(ContextKeyAuthFailed).(*bool); ok && *value {
	// 	return AuthFailed
	// }
	// username := GetUsernameFromSSHCtx(ctx)
	// res = AuthFailed
	// client, ok := ctx.Value(ContextKeyClient).(*UserAuthClient)
	// if !ok {
	// 	logger.Errorf("SSH conn[%s] user %s Mfa Auth failed: not found session client.",
	// 		ctx.SessionID(), username)
	// 	return
	// }
	// status, ok2 := ctx.Value(ContextKeyAuthStatus).(StatusAuth)
	// if !ok2 {
	// 	logger.Errorf("SSH conn[%s] user %s unknown auth", ctx.SessionID(), username)
	// 	return
	// }
	// var checkAuth func(ssh.Context, gossh.KeyboardInteractiveChallenge) bool
	// switch status {
	// case authConfirmRequired:
	// 	checkAuth = client.CheckConfirmAuth
	// case authMFARequired:
	// 	checkAuth = client.CheckMFAAuth
	// }
	// if checkAuth != nil && checkAuth(ctx, challenger) {
	// 	res = AuthSuccessful
	// }
	// return
}

// func GetUsernameFromSSHCtx(ctx ssh.Context) string {
// 	if directReq, ok := ctx.Value(ContextKeyDirectLoginFormat).(*DirectLoginAssetReq); ok {
// 		return directReq.Username
// 	}
// 	username := ctx.User()
// 	if req, ok := ParseDirectUserFormat(username); ok {
// 		username = req.Username
// 		ctx.SetValue(ContextKeyDirectLoginFormat, &req)
// 	}
// 	return username
// }
