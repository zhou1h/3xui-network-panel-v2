package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/zhou1h/3xui-network-panel-v2/ent"
	"github.com/zhou1h/3xui-network-panel-v2/ent/job"
	"github.com/zhou1h/3xui-network-panel-v2/ent/resource"
	"github.com/zhou1h/3xui-network-panel-v2/ent/user"
	appcore "github.com/zhou1h/3xui-network-panel-v2/internal/app"
	"github.com/zhou1h/3xui-network-panel-v2/internal/security"
)

type Server struct{ app *appcore.App }

func New(app *appcore.App) *gin.Engine {
	s := &Server{app: app}
	r := gin.New()
	r.Use(gin.Recovery(), securityHeaders())
	r.GET("/api/v1/health", s.health)
	r.POST("/api/v1/auth/login", s.login)
	auth := r.Group("/api/v1", s.requireAuth())
	auth.GET("/auth/me", s.me)
	auth.POST("/auth/logout", s.logout)
	auth.GET("/dashboard", s.dashboard)
	auth.GET("/resources", s.listResources)
	auth.POST("/resources", s.createResource)
	auth.PUT("/resources/:id", s.updateResource)
	auth.DELETE("/resources/:id", s.deleteResource)
	auth.GET("/jobs", s.listJobs)
	return r
}

func securityHeaders() gin.HandlerFunc { return func(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Referrer-Policy", "same-origin")
	c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	c.Next()
} }

func (s *Server) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second); defer cancel()
	if err := s.app.Ready(ctx); err != nil { c.JSON(http.StatusServiceUnavailable, gin.H{"status":"degraded"}); return }
	c.JSON(http.StatusOK, gin.H{"status":"ok", "time":time.Now().UTC()})
}

func (s *Server) login(c *gin.Context) {
	var input struct{ Username string `json:"username"`; Password string `json:"password"` }
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_request"}); return }
	account, err := s.app.DB.User.Query().Where(user.UsernameEQ(strings.TrimSpace(input.Username)), user.EnabledEQ(true)).Only(c)
	if err != nil || !security.VerifyPassword(account.PasswordHash, input.Password) { time.Sleep(350*time.Millisecond); c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid_credentials"}); return }
	token, err := security.RandomToken(32); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"internal_error"}); return }
	csrf, err := security.RandomToken(24); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"internal_error"}); return }
	key := "session:" + token
	if err := s.app.Redis.HSet(c, key, "user_id", account.ID, "csrf", csrf).Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"session_failed"}); return }
	s.app.Redis.Expire(c, key, s.app.Config.SessionTTL)
	http.SetCookie(c.Writer, &http.Cookie{Name:"panel_session", Value:token, Path:"/", MaxAge:int(s.app.Config.SessionTTL.Seconds()), HttpOnly:true, Secure:s.app.Config.CookieSecure, SameSite:http.SameSiteStrictMode})
	c.JSON(http.StatusOK, gin.H{"user": publicUser(account), "csrfToken":csrf})
}

func (s *Server) logout(c *gin.Context) {
	if token, err := c.Cookie("panel_session"); err == nil { s.app.Redis.Del(c, "session:"+token) }
	http.SetCookie(c.Writer, &http.Cookie{Name:"panel_session", Value:"", Path:"/", MaxAge:-1, HttpOnly:true, Secure:s.app.Config.CookieSecure, SameSite:http.SameSiteStrictMode})
	c.JSON(http.StatusOK, gin.H{"ok":true})
}

func (s *Server) me(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"user":publicUser(c.MustGet("user").(*ent.User)), "csrfToken":c.GetString("csrf")}) }

func (s *Server) requireAuth() gin.HandlerFunc { return func(c *gin.Context) {
	token, err := c.Cookie("panel_session"); if err != nil || token == "" { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	values, err := s.app.Redis.HGetAll(c, "session:"+token).Result(); if err != nil || len(values) == 0 { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	uid, err := strconv.Atoi(values["user_id"]); if err != nil { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	account, err := s.app.DB.User.Get(c, uid); if err != nil || !account.Enabled { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.GetHeader("X-CSRF-Token") != values["csrf"] { c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error":"csrf_failed"}); return }
	c.Set("user", account); c.Set("csrf", values["csrf"]); c.Next()
} }

func publicUser(u *ent.User) gin.H { return gin.H{"id":u.ID,"username":u.Username,"role":u.Role,"mustChangePassword":u.MustChangePassword} }

func (s *Server) dashboard(c *gin.Context) {
	resources, _ := s.app.DB.Resource.Query().Count(c)
	online, _ := s.app.DB.Resource.Query().Where(resource.StatusEQ("ok")).Count(c)
	queued, _ := s.app.DB.Job.Query().Where(job.StatusIn("queued","running")).Count(c)
	clients, _ := s.app.DB.Resource.Query().Aggregate(ent.Sum(resource.FieldClientCount)).Int(c)
	nodes, _ := s.app.DB.Resource.Query().Aggregate(ent.Sum(resource.FieldNodeCount)).Int(c)
	socks, _ := s.app.DB.Resource.Query().Aggregate(ent.Sum(resource.FieldSocksCount)).Int(c)
	c.JSON(http.StatusOK, gin.H{"resources":resources,"online":online,"activeJobs":queued,"clients":clients,"nodes":nodes,"socks5":socks})
}

type resourceInput struct { Code string `json:"code"`; Name string `json:"name"`; Host string `json:"host"`; ManagementURL string `json:"managementUrl"`; AccessToken string `json:"accessToken"` }

func (s *Server) listResources(c *gin.Context) {
	items, err := s.app.DB.Resource.Query().Order(ent.Asc(resource.FieldCode)).All(c); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"query_failed"}); return }
	result := make([]gin.H, 0, len(items)); for _, item := range items { result = append(result, resourceJSON(item)) }
	c.JSON(http.StatusOK, gin.H{"items":result})
}

func (s *Server) createResource(c *gin.Context) {
	var input resourceInput; if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_request"}); return }
	if strings.TrimSpace(input.Code)=="" || strings.TrimSpace(input.Name)=="" || strings.TrimSpace(input.Host)=="" { c.JSON(http.StatusUnprocessableEntity, gin.H{"error":"required_fields"}); return }
	token, err := s.app.Cipher.Encrypt(input.AccessToken); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"encrypt_failed"}); return }
	item, err := s.app.DB.Resource.Create().SetCode(strings.ToUpper(strings.TrimSpace(input.Code))).SetName(strings.TrimSpace(input.Name)).SetHost(strings.TrimSpace(input.Host)).SetManagementURL(strings.TrimSpace(input.ManagementURL)).SetAccessTokenCiphertext(token).Save(c)
	if err != nil { c.JSON(http.StatusConflict, gin.H{"error":"resource_exists"}); return }
	s.audit(c,"resource.create","resource",strconv.Itoa(item.ID)); c.JSON(http.StatusCreated, resourceJSON(item))
}

func (s *Server) updateResource(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
	var input resourceInput; if err := c.ShouldBindJSON(&input); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_request"}); return }
	update := s.app.DB.Resource.UpdateOneID(id).SetCode(strings.ToUpper(strings.TrimSpace(input.Code))).SetName(strings.TrimSpace(input.Name)).SetHost(strings.TrimSpace(input.Host)).SetManagementURL(strings.TrimSpace(input.ManagementURL))
	if input.AccessToken != "" { token, err := s.app.Cipher.Encrypt(input.AccessToken); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"encrypt_failed"}); return }; update.SetAccessTokenCiphertext(token) }
	item, err := update.Save(c); if err != nil { c.JSON(http.StatusNotFound, gin.H{"error":"resource_not_found"}); return }
	s.audit(c,"resource.update","resource",strconv.Itoa(item.ID)); c.JSON(http.StatusOK, resourceJSON(item))
}

func (s *Server) deleteResource(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
	if err := s.app.DB.Resource.DeleteOneID(id).Exec(c); err != nil { c.JSON(http.StatusNotFound, gin.H{"error":"resource_not_found"}); return }
	s.audit(c,"resource.delete","resource",strconv.Itoa(id)); c.JSON(http.StatusOK, gin.H{"ok":true})
}

func resourceJSON(item *ent.Resource) gin.H { return gin.H{"id":item.ID,"code":item.Code,"name":item.Name,"host":item.Host,"managementUrl":item.ManagementURL,"hasAccessToken":item.AccessTokenCiphertext!="","status":item.Status,"nodeCount":item.NodeCount,"clientCount":item.ClientCount,"socksCount":item.SocksCount,"latencyMs":item.LatencyMs,"lastCheckedAt":item.LastCheckedAt} }

func (s *Server) listJobs(c *gin.Context) { items, err := s.app.DB.Job.Query().Order(ent.Desc(job.FieldCreatedAt)).Limit(100).All(c); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"query_failed"}); return }; c.JSON(http.StatusOK, gin.H{"items":items}) }

func (s *Server) audit(c *gin.Context, action, targetType, targetID string) {
	account, _ := c.Get("user"); uid := 0; if account != nil { uid = account.(*ent.User).ID }
	_ = s.app.DB.AuditLog.Create().SetUserID(uid).SetAction(action).SetTargetType(targetType).SetTargetID(targetID).SetIP(c.ClientIP()).Exec(c)
}

func randomID() string { b:=make([]byte,12); _,_=rand.Read(b); return base64.RawURLEncoding.EncodeToString(b) }

var _ = errors.Is
var _ = redis.Nil
var _ = randomID
