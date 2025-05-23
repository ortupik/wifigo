package router

import (
	"github.com/gin-gonic/gin"

	storage "github.com/ortupik/wifigo/badger"
	gconfig "github.com/ortupik/wifigo/config"
	gcontroller "github.com/ortupik/wifigo/controller"
	glib "github.com/ortupik/wifigo/lib"
	gmiddleware "github.com/ortupik/wifigo/lib/middleware"
	queue "github.com/ortupik/wifigo/queue"
	"github.com/ortupik/wifigo/server/controller"
	gservice "github.com/ortupik/wifigo/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	mikrotik "github.com/ortupik/wifigo/mikrotik"
	handler "github.com/ortupik/wifigo/server/handler"
	"github.com/ortupik/wifigo/websocket"
)

var (
	mikrotikController   *controller.MikroTikController
	mpesaCallbackHandler *handler.MpesaCallbackHandler
	mpesaController      *controller.MpesaController // Use the correct controller package
)

// SetupRouter sets up all the routes
func SetupRouter(configure *gconfig.Configuration, store *storage.Store,
	manager *mikrotik.Manager, queueClient *queue.Client, wsHub *websocket.Hub) (*gin.Engine, error) {
	// Set Gin mode based on environment
	if gconfig.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Static("/static", "./static")

	// Load HTML templates
	r.LoadHTMLGlob("templates/*.html")

	r.GET("/checkout", controller.CheckoutController)
	r.GET("/howto", controller.HowtoController)
	r.GET("/confirm", controller.ConfirmController)

	// Setup session middleware
	cookieStore := cookie.NewStore([]byte("a1b2c3d4e5f678901234567890abcdef0123456789abcdef0123456789abcdef")) // Use session secret from config(configure.Auth.SessionSecret)
	r.Use(sessions.Sessions("wifigo_session", cookieStore))

	r.GET("/ws", func(c *gin.Context) {
		wsHub.HandleWebSocket(c.Writer, c.Request)
	})

	// Initialize handlers and controllers
	mpesaCallbackHandler = handler.NewMpesaCallbackHandler(queueClient, wsHub)
	mpesaController = controller.NewMpesaController()
	mikrotikController = controller.NewMikroTikController(manager)

	// Disable trusted proxies for security unless specifically configured
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

	// Set trusted platform for client IP detection
	trustedPlatform := configure.Security.TrustedPlatform
	switch trustedPlatform {
	case "cf":
		r.TrustedPlatform = gin.PlatformCloudflare
	case "google":
		r.TrustedPlatform = gin.PlatformGoogleAppEngine
	default:
		r.TrustedPlatform = trustedPlatform
	}

	// Apply security middleware in proper order (most critical first)

	// 1. WAF (should be first to block malicious requests)
	if gconfig.IsWAF() {
		r.Use(gmiddleware.Firewall(
			configure.Security.Firewall.ListType,
			configure.Security.Firewall.IP,
		))
	}

	// 2. CORS (handle preflight requests early)
	if gconfig.IsCORS() {
		r.Use(gmiddleware.CORS(configure.Security.CORS))
	}

	// 3. Origin check
	if gconfig.IsOriginCheck() {
		r.Use(gmiddleware.CheckOrigin())
	}

	// 4. Rate limiting (after basic security checks, before heavy processing)
	if gconfig.IsRateLimit() {
		limiterInstance, err := glib.InitRateLimiter(
			configure.Security.RateLimit,
			trustedPlatform,
		)
		if err != nil {
			return nil, err
		}
		r.Use(gmiddleware.RateLimit(limiterInstance))
	}

	// 5. Sentry for error reporting (after security middleware to avoid reporting blocked requests)
	if gconfig.IsSentry() {
		if _, err := gmiddleware.InitSentry(
			configure.Logger.SentryDsn,
			configure.Server.ServerEnv,
			configure.Version,
			configure.Logger.PerformanceTracing,
			configure.Logger.TracesSampleRate,
		); err != nil {
			return nil, err
		}
		r.Use(gmiddleware.SentryCapture())
	}

	// 6. View rendering (last middleware to apply)
	if gconfig.IsTemplatingEngine() {
		r.Use(gmiddleware.Pongo2(configure.ViewConfig.Directory))
	}

	// API Status endpoint
	r.GET("", controller.APIStatus)

	// Register all API routes
	registerAPIRoutes(r, configure, mikrotikController, mpesaController, mpesaCallbackHandler)

	return r, nil
}

// registerAPIRoutes sets up all API routes
func registerAPIRoutes(r *gin.Engine, configure *gconfig.Configuration, mikrotikController *controller.MikroTikController, mpesaController *controller.MpesaController, mpesaCallbackHandler *handler.MpesaCallbackHandler) {
	v1 := r.Group("/api/v1/")

	// Public routes (no authentication required)
	registerPublicRoutes(v1, configure)

	// Protected routes (authentication required)
	if gconfig.IsRDBMS() {
		registerAuthRoutes(v1, configure)
		registerUserRoutes(v1, configure)
		registerResourceRoutes(v1, configure)
		registerHotspotRoutes(v1, configure)
		registerMikrotikRoutes(v1, configure)
		registerMpesaRoutes(v1, configure)
	}

	// Playground routes for development and testing
	registerPlaygroundRoutes(v1, configure)

	// Basic Auth demo routes
	registerBasicAuthRoutes(v1, configure)

	// QueryString demo routes
	v1.GET("query/*q", controller.QueryString)
}

// registerPublicRoutes sets up routes that don't require authentication
func registerPublicRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	if gconfig.IsRDBMS() {
		// Authentication endpoints
		v1.POST("register", gcontroller.CreateUserAuth)
		v1.POST("login", gcontroller.Login)

		// Email verification endpoints
		if gconfig.IsEmailVerificationService() && gconfig.IsRedis() {
			v1.POST("verify", gcontroller.VerifyEmail)
			v1.POST("resend-verification-email", gcontroller.CreateVerificationEmail)
			v1.POST("verify-updated-email", gcontroller.VerifyUpdatedEmail)
		}

		// Password recovery endpoints
		if gconfig.IsEmailService() {
			passGroup := v1.Group("password")
			passGroup.POST("forgot", gcontroller.PasswordForgot)
			passGroup.POST("reset", gcontroller.PasswordRecover)
		}
	}
}

// registerAuthRoutes sets up authentication-related routes
func registerAuthRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	// Logout endpoint
	logoutGroup := v1.Group("logout")
	logoutGroup.Use(gmiddleware.JWT()).
		Use(gmiddleware.RefreshJWT()).
		Use(gservice.JWTBlacklistChecker())
	logoutGroup.POST("", gcontroller.Logout)

	// Token refresh endpoint
	refreshGroup := v1.Group("refresh")
	refreshGroup.Use(gmiddleware.JWT()).
		Use(gmiddleware.RefreshJWT()).
		Use(gservice.JWTBlacklistChecker())
	refreshGroup.POST("", gcontroller.Refresh)

	// 2FA endpoints
	if gconfig.Is2FA() {
		twoFAGroup := v1.Group("2fa")
		twoFAGroup.Use(gmiddleware.JWT()).
			Use(gservice.JWTBlacklistChecker())

		// Initial setup endpoints (no 2FA required)
		twoFAGroup.POST("setup", gcontroller.Setup2FA)
		twoFAGroup.POST("activate", gcontroller.Activate2FA)
		twoFAGroup.POST("validate", gcontroller.Validate2FA)
		twoFAGroup.POST("validate-backup-code", gcontroller.ValidateBackup2FA)

		// Operations requiring 2FA verification
		twoFAProtected := twoFAGroup.Group("")
		twoFAProtected.Use(gmiddleware.TwoFA(
			configure.Security.TwoFA.Status.On,
			configure.Security.TwoFA.Status.Off,
			configure.Security.TwoFA.Status.Verified,
		))
		twoFAProtected.POST("create-backup-codes", gcontroller.CreateBackup2FA)
		twoFAProtected.POST("deactivate", gcontroller.Deactivate2FA)
	}
}

// createAuthMiddleware returns middleware chain for protected routes
func createAuthMiddleware(configure *gconfig.Configuration) []gin.HandlerFunc {
	middleware := []gin.HandlerFunc{
		gmiddleware.JWT(),
		gservice.JWTBlacklistChecker(),
	}

	if gconfig.Is2FA() {
		middleware = append(middleware, gmiddleware.TwoFA(
			configure.Security.TwoFA.Status.On,
			configure.Security.TwoFA.Status.Off,
			configure.Security.TwoFA.Status.Verified,
		))
	}

	return middleware
}

// registerUserRoutes sets up user-related routes
func registerUserRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	// Password management
	passGroup := v1.Group("password")
	passGroup.Use(createAuthMiddleware(configure)...)
	passGroup.POST("edit", gcontroller.PasswordUpdate)

	// Email management
	emailGroup := v1.Group("email")
	emailGroup.Use(createAuthMiddleware(configure)...)
	emailGroup.POST("update", gcontroller.UpdateEmail)
	emailGroup.GET("unverified", gcontroller.GetUnverifiedEmail)
	emailGroup.POST("resend-verification-email", gcontroller.ResendVerificationCodeToModifyActiveEmail)

	// User CRUD operations
	userGroup := v1.Group("users")
	userGroup.Use(createAuthMiddleware(configure)...)
	userGroup.GET("", controller.GetUsers)
	userGroup.GET("/:id", controller.GetUser)
	userGroup.POST("", controller.CreateUser)
	userGroup.PUT("", controller.UpdateUser)
}
func registerMpesaRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	mpesaGroup := v1.Group("mpesa")
	mpesaGroup.POST("/checkout", mpesaController.ExpressStkHandler)
	mpesaGroup.GET("/transaction", mpesaController.GetTransactionStatus)
	mpesaGroup.POST("/callback", mpesaCallbackHandler.MpesaStkHandlerCallback)
	mpesaGroup.Use(createAuthMiddleware(configure)...)
}

// registerResourceRoutes sets up resource-related routes
func registerResourceRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	// Test JWT endpoint
	testJWTGroup := v1.Group("test-jwt")
	testJWTGroup.Use(createAuthMiddleware(configure)...)
}

// registerHotspotRoutes sets up hotspot-related routes
func registerHotspotRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	hotspotUsers := v1.Group("hotspot/users")
	hotspotUsers.Use(createAuthMiddleware(configure)...)

	// Hotspot user CRUD
	hotspotUsers.POST("/", controller.CreateHotspotUser)
	hotspotUsers.GET("/:username", controller.GetHotspotUser)
	hotspotUsers.PUT("/:username", controller.UpdateHotspotUser)
	hotspotUsers.DELETE("/:username", controller.DeleteHotspotUser)

	// Hotspot user attributes and groups
	hotspotUsers.POST("/:username/check", controller.AddOrUpdateRadCheckAttribute)
	hotspotUsers.DELETE("/:username/check/:attribute", controller.DeleteRadCheckAttribute)
	hotspotUsers.POST("/:username/reply", controller.AddOrUpdateRadReplyAttribute)
	hotspotUsers.DELETE("/:username/reply/:attribute", controller.DeleteRadReplyAttribute)
	hotspotUsers.POST("/:username/group", controller.AddRadUserGroup)
	hotspotUsers.DELETE("/:username/group/:groupname", controller.DeleteRadUserGroup)
}

func registerMikrotikRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	// Create mikrotik API group
	mikrotikAPI := v1.Group("/mikrotik")
	mikrotikAPI.Use(createAuthMiddleware(configure)...)

	// Device CRUD routes
	mikrotikAPI.GET("/devices", mikrotikController.GetDevices)
	mikrotikAPI.POST("/devices", mikrotikController.CreateDevice)
	mikrotikAPI.GET("/devices/:id", mikrotikController.GetDevice)
	mikrotikAPI.PUT("/devices/:id", mikrotikController.UpdateDevice)
	mikrotikAPI.DELETE("/devices/:id", mikrotikController.DeleteDevice)

	// Device status routes
	mikrotikAPI.PATCH("/devices/:id/status", mikrotikController.UpdateDeviceStatus)
	mikrotikAPI.GET("/devices/status/:status", mikrotikController.GetDevicesByStatus)

	// Device statistics and utilities
	mikrotikAPI.GET("/devices/stats", mikrotikController.GetDeviceStats)
	mikrotikAPI.POST("/devices/:id/test", mikrotikController.TestDeviceConnection)
}

// registerPlaygroundRoutes sets up development and testing routes
func registerPlaygroundRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	// Redis playground
	if gconfig.IsRedis() {
		redisGroup := v1.Group("playground")
		// String operations
		redisGroup.GET("/redis_read", controller.RedisRead)
		redisGroup.POST("/redis_create", controller.RedisCreate)
		redisGroup.DELETE("/redis_delete", controller.RedisDelete)
		// Hash operations
		redisGroup.GET("/redis_read_hash", controller.RedisReadHash)
		redisGroup.POST("/redis_create_hash", controller.RedisCreateHash)
		redisGroup.DELETE("/redis_delete_hash", controller.RedisDeleteHash)
	}

	// MongoDB playground
	if gconfig.IsMongo() {
		mongoGroup := v1.Group("playground-mongo")
		mongoGroup.POST("/mongo_create_one", controller.MongoCreateOne)
		mongoGroup.GET("/mongo_get_all", controller.MongoGetAll)
		mongoGroup.GET("/mongo_get_by_id/:id", controller.MongoGetByID)
		mongoGroup.POST("/mongo_get_by_filter", controller.MongoGetByFilter)
		mongoGroup.PUT("/mongo_update_by_id", controller.MongoUpdateByID)
		mongoGroup.DELETE("/mongo_delete_field_by_id", controller.MongoDeleteFieldByID)
		mongoGroup.DELETE("/mongo_delete_doc_by_id/:id", controller.MongoDeleteByID)
	}
}

// registerBasicAuthRoutes sets up basic auth protected routes
func registerBasicAuthRoutes(v1 *gin.RouterGroup, configure *gconfig.Configuration) {
	if gconfig.IsBasicAuth() {
		user := configure.Security.BasicAuth.Username
		pass := configure.Security.BasicAuth.Password

		basicAuthGroup := v1.Group("access_resources")
		basicAuthGroup.Use(gin.BasicAuth(gin.Accounts{user: pass}))
	}
}
