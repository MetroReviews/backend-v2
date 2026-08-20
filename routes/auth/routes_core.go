package auth

import (
	"github.com/MetroReviews/backend-v2/types"
	"github.com/go-chi/chi/v5"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func registerRoutes(r *chi.Mux) {
	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/auth/login",
		OpId:    "auth_login",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Login",
				Description: "Verifies a Discord access token against Discord and mints a Metro session token for it, creating the Metro user on first login. Intended for first-party clients that run their own Discord OAuth2 flow.",
				Req:         loginRequest{},
				Resp:        loginResponse{},
			}
		},
		Handler: login,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/auth/register",
		OpId:    "auth_register",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Register",
				Description: "Creates a new Metro user with an email/password login and mints a session for it. Fails with 409 if the email is already registered.",
				Req:         registerRequest{},
				Resp:        localAuthResponse{},
			}
		},
		Handler: register,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/auth/login/password",
		OpId:    "auth_login_password",
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Login (Email/Password)",
				Description: "Verifies an email/password against a registered local account and mints a Metro session token for it.",
				Req:         loginPasswordRequest{},
				Resp:        localAuthResponse{},
			}
		},
		Handler: loginPassword,
	}.Route(r)

	uapi.Route{
		Method:  uapi.POST,
		Pattern: "/auth/password",
		OpId:    "auth_set_password",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Set Password",
				Description: "Sets or replaces the email/password login on the caller's own account — the same account they're already logged into (e.g. via Discord), letting either method log in afterward. Requires a logged-in user session. Fails with 409 if the email is already registered to a different account.",
				Req:         setPasswordRequest{},
				Resp:        types.ApiError{},
			}
		},
		Handler: setPassword,
	}.Route(r)

	uapi.Route{
		Method:  uapi.GET,
		Pattern: "/me",
		OpId:    "get_me",
		Auth:    []uapi.AuthType{{Type: "User"}},
		Docs: func() *docs.Doc {
			return &docs.Doc{
				Summary:     "Get My Profile",
				Description: "Returns the calling user's own profile: username, avatar, bio, is_staff/banned flags, and linked Discord ID. This is the normal 'who am I' endpoint; pair it with GET /me/permissions for roles/permissions.",
				Resp:        types.User{},
			}
		},
		Handler: getMe,
	}.Route(r)
}
