package handlers

import (
    "log/slog"

    "github.com/cfilipov/dockge/backend-go/internal/models"
    "github.com/cfilipov/dockge/backend-go/internal/ws"
)

func RegisterAuthHandlers(app *App) {
    app.WS.Handle("login", app.handleLogin)
    app.WS.Handle("loginByToken", app.handleLoginByToken)
    app.WS.Handle("logout", app.handleLogout)
    app.WS.Handle("setup", app.handleSetup)
    app.WS.Handle("changePassword", app.handleChangePassword)
    app.WS.Handle("getTurnstileSiteKey", app.handleGetTurnstileSiteKey)
    app.WS.Handle("needSetup", app.handleNeedSetup)

    // 2FA stubs — not implemented yet
    app.WS.Handle("prepare2FA", app.handleStub2FA)
    app.WS.Handle("save2FA", app.handleStub2FA)
    app.WS.Handle("disable2FA", app.handleStub2FA)
    app.WS.Handle("verifyToken", app.handleStub2FA)
    app.WS.Handle("twoFAStatus", app.handleTwoFAStatus)

    app.WS.HandleConnect(func(c *ws.Conn) {
        // Send server info on every new connection
        c.SendEvent("info", map[string]interface{}{
            "version":       app.Version,
            "latestVersion": app.Version,
            "isContainer":   true,
        })

        // If no users exist, tell the client to show the setup page
        if app.NeedSetup {
            c.SendEvent("setup")
        }
    })
}

func (app *App) handleLogin(c *ws.Conn, msg *ws.ClientMessage) {
    args := parseArgs(msg)
    if len(args) == 0 {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Invalid arguments"})
        }
        return
    }

    // Login args can be either positional [username, password, token, captchaToken]
    // or an object {username, password, captchaToken}
    var username, password string

    // Try object format first
    var loginData struct {
        Username     string `json:"username"`
        Password     string `json:"password"`
        CaptchaToken string `json:"captchaToken"`
    }
    if argObject(args, 0, &loginData) && loginData.Username != "" {
        username = loginData.Username
        password = loginData.Password
    } else {
        // Positional format
        username = argString(args, 0)
        password = argString(args, 1)
    }

    if username == "" || password == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "authIncorrectCreds", MsgI18n: true})
        }
        return
    }

    user, err := app.Users.FindByUsername(username)
    if err != nil {
        slog.Error("login lookup", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Internal error"})
        }
        return
    }

    if user == nil || !models.VerifyPassword(password, user.Password) {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "authIncorrectCreds", MsgI18n: true})
        }
        return
    }

    token, err := models.CreateJWT(user, app.JWTSecret)
    if err != nil {
        slog.Error("create jwt", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Internal error"})
        }
        return
    }

    c.SetUser(user.ID)
    app.afterLogin(c)

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true, Token: token})
    }

    slog.Info("user logged in", "username", username)
}

func (app *App) handleLoginByToken(c *ws.Conn, msg *ws.ClientMessage) {
    args := parseArgs(msg)
    token := argString(args, 0)
    if token == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "authInvalidToken", MsgI18n: true})
        }
        return
    }

    claims, err := models.VerifyJWT(token, app.JWTSecret)
    if err != nil {
        slog.Debug("token verify failed", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "authInvalidToken", MsgI18n: true})
        }
        return
    }

    user, err := app.Users.FindByUsername(claims.Username)
    if err != nil {
        slog.Error("token user lookup", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Internal error"})
        }
        return
    }

    if user == nil {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "authUserInactiveOrDeleted", MsgI18n: true})
        }
        return
    }

    // Password change detection: compare shake256(storedPassword) with token's h claim
    if claims.H != models.Shake256Hex(user.Password, 16) {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "authInvalidToken", MsgI18n: true})
        }
        return
    }

    c.SetUser(user.ID)
    app.afterLogin(c)

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }

    slog.Debug("token login", "username", claims.Username)
}

func (app *App) handleSetup(c *ws.Conn, msg *ws.ClientMessage) {
    args := parseArgs(msg)
    username := argString(args, 0)
    password := argString(args, 1)

    if username == "" || password == "" {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Username and password required"})
        }
        return
    }

    if len(password) < 6 {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Password is too weak. It should be at least 6 characters."})
        }
        return
    }

    // Check no users exist
    count, err := app.Users.Count()
    if err != nil {
        slog.Error("setup count", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Internal error"})
        }
        return
    }
    if count > 0 {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Dockge has already been set up"})
        }
        return
    }

    _, err = app.Users.Create(username, password)
    if err != nil {
        slog.Error("setup create user", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to create user"})
        }
        return
    }

    app.NeedSetup = false

    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":      true,
            "msg":     "successAdded",
            "msgi18n": true,
        })
    }

    slog.Info("setup complete", "username", username)
}

func (app *App) handleChangePassword(c *ws.Conn, msg *ws.ClientMessage) {
    uid := checkLogin(c, msg)
    if uid == 0 {
        return
    }

    args := parseArgs(msg)
    var data struct {
        CurrentPassword string `json:"currentPassword"`
        NewPassword     string `json:"newPassword"`
    }
    if !argObject(args, 0, &data) {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Invalid arguments"})
        }
        return
    }

    // Verify current password
    user, err := app.Users.FindByID(uid)
    if err != nil || user == nil {
        slog.Error("change password lookup", "err", err, "uid", uid)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Internal error"})
        }
        return
    }
    if !models.VerifyPassword(data.CurrentPassword, user.Password) {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "authIncorrectCreds", MsgI18n: true})
        }
        return
    }

    if len(data.NewPassword) < 6 {
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Password too weak"})
        }
        return
    }

    if err := app.Users.ChangePassword(uid, data.NewPassword); err != nil {
        slog.Error("change password", "err", err)
        if msg.ID != nil {
            c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "Failed to change password"})
        }
        return
    }

    // Disconnect all other sessions so they must re-auth with new password
    app.WS.BroadcastAuthenticated("refresh")

    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true, Msg: "Password changed"})
    }
}

func (app *App) handleGetTurnstileSiteKey(c *ws.Conn, msg *ws.ClientMessage) {
    // Turnstile/CAPTCHA not configured in self-hosted mode
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

func (app *App) handleNeedSetup(c *ws.Conn, msg *ws.ClientMessage) {
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":        true,
            "needSetup": app.NeedSetup,
        })
    }
}

func (app *App) handleLogout(c *ws.Conn, msg *ws.ClientMessage) {
    c.SetUser(0)
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.OkResponse{OK: true})
    }
}

// handleStub2FA returns a "not supported" error for 2FA operations.
func (app *App) handleStub2FA(c *ws.Conn, msg *ws.ClientMessage) {
    if msg.ID != nil {
        c.SendAck(*msg.ID, ws.ErrorResponse{OK: false, Msg: "2FA is not yet supported"})
    }
}

// handleTwoFAStatus returns that 2FA is not enabled.
func (app *App) handleTwoFAStatus(c *ws.Conn, msg *ws.ClientMessage) {
    if msg.ID != nil {
        c.SendAck(*msg.ID, map[string]interface{}{
            "ok":     true,
            "status": false,
        })
    }
}

// afterLogin sends initial data to a freshly authenticated connection.
func (app *App) afterLogin(c *ws.Conn) {
    // Send server info
    c.SendEvent("info", map[string]interface{}{
        "version":       app.Version,
        "latestVersion": app.Version,
        "isContainer":   true,
    })

    // NOTE: Do NOT send "autoLogin" here. That event is only for when auth is
    // disabled (every connection is auto-authenticated). Sending it after a real
    // login causes the frontend to overwrite the JWT token with "autoLogin",
    // breaking token-based re-auth on subsequent page loads.

    // Send agent list
    agents, err := app.Agents.GetAll()
    if err != nil {
        slog.Error("get agents", "err", err)
        agents = nil
    }
    agentMap := make(map[string]interface{})
    for _, a := range agents {
        agentMap[a.URL] = map[string]interface{}{
            "url":    a.URL,
            "name":   a.Name,
            "active": a.Active,
        }
    }
    c.SendEvent("agentList", agentMap)

    // Send cached stack list immediately so the UI populates instantly
    app.sendStackListTo(c)
}
