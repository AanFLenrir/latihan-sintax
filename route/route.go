package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"latihan-fiber/app/service"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
)

type Dependencies struct {
	Pool                *pgxpool.Pool
	Mongo               *mongo.Client
	JWT                 *helper.JWTManager
	Permissions         *helper.PermissionSet
	UserService         *service.UserService
	AuthService         *service.AuthService
	CourseService       *service.CourseService
	SupervisorService   *service.SupervisorService
	PortfolioService    *service.PortfolioService
	ReviewService       *service.ReviewService
	NotificationService *service.NotificationService
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	api.Get("/health", healthCheck(deps.Pool, deps.Mongo))

	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	users := api.Group("/users",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	perms := deps.Permissions

	users.Get("/",
		middleware.RequirePermission(perms, "user:list"),
		deps.UserService.List)
	users.Post("/",
		middleware.RequirePermission(perms, "user:update:any"),
		deps.UserService.Create)
	users.Delete("/:id",
		middleware.RequirePermission(perms, "user:delete"),
		deps.UserService.Delete)
	users.Patch("/:id/role",
		middleware.RequirePermission(perms, "role:assign"),
		deps.UserService.AssignRole)
	users.Get("/:id", deps.UserService.Get)
	users.Put("/:id", deps.UserService.Replace)
	users.Patch("/:id", deps.UserService.Patch)

	// --- kelas dan KRS ---
	courses := api.Group("/courses",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))

	courses.Get("/", deps.CourseService.List)
	courses.Get("/mine", deps.CourseService.Mine)
	courses.Post("/swap", deps.CourseService.Swap)
	courses.Get("/:id", deps.CourseService.Get)
	courses.Post("/:id/enroll", deps.CourseService.Enroll)
	courses.Delete("/:id/enroll", deps.CourseService.Drop)
	courses.Patch("/:id", middleware.RequirePermission(perms, "course:manage"), deps.CourseService.UpdateQuota)
	courses.Patch("/:id/quota", middleware.RequirePermission(perms, "course:manage"), deps.CourseService.UpdateQuota)

	if deps.ReviewService != nil {
		courses.Get("/:id/reviews", deps.ReviewService.ListByCourse)
		courses.Get("/:id/reviews/summary", deps.ReviewService.Summary)
		courses.Post("/:id/reviews", deps.ReviewService.Create)
	}

	enrollments := api.Group("/enrollments",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	enrollments.Post("/swap", deps.CourseService.Swap)

	// --- pembimbing tugas akhir (Tugas Mandiri D.2) ---
	if deps.SupervisorService != nil {
		supervisors := api.Group("/supervisors",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT))

		supervisors.Get("/", deps.SupervisorService.List)
		supervisors.Get("/:id", deps.SupervisorService.Get)
		supervisors.Post("/:id/assign",
			middleware.RequirePermission(perms, "supervision:manage"),
			deps.SupervisorService.Assign)
		supervisors.Patch("/:id",
			middleware.RequirePermission(perms, "supervision:manage"),
			deps.SupervisorService.UpdateQuota)

		supervisions := api.Group("/supervisions",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT))
		supervisions.Post("/switch",
			middleware.RequirePermission(perms, "supervision:manage"),
			deps.SupervisorService.Switch)
		supervisors.Post("/switch",
			middleware.RequirePermission(perms, "supervision:manage"),
			deps.SupervisorService.Switch)
	}

	// --- ulasan kelas (MongoDB) Langkah 8 ---
	if deps.ReviewService != nil {
		reviews := api.Group("/reviews",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT))
		reviews.Get("/:id", deps.ReviewService.Get)
		reviews.Patch("/:id", deps.ReviewService.Update)
		reviews.Delete("/:id", deps.ReviewService.Delete)
		reviews.Post("/:id/replies", deps.ReviewService.Reply)
		reviews.Post("/:id/helpful", deps.ReviewService.MarkHelpful)
	}

	// --- portofolio mahasiswa (MongoDB + PostgreSQL) ---
	if deps.PortfolioService != nil {
		// Aggregation endpoint - statistik 10 teknologi terpopuler
		api.Get("/portfolios/stats/technologies", deps.PortfolioService.GetTechnologyStats)

		students := api.Group("/students",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT))

		students.Get("/:id/portfolio", deps.PortfolioService.Get)
		students.Put("/:id/portfolio", deps.PortfolioService.Upsert)
		students.Post("/:id/portfolio/projects", deps.PortfolioService.AddProject)
		students.Delete("/:id/portfolio/projects/:projectId", deps.PortfolioService.DeleteProject)
	}

	// --- notifikasi (diisi oleh worker, dibaca lewat API) ---
	if deps.NotificationService != nil {
		notifications := api.Group("/notifications",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT))
		notifications.Get("/mine", deps.NotificationService.Mine)

		admin := api.Group("/admin",
			middleware.RequireJSON,
			middleware.RequireAuth(deps.JWT))
		admin.Get("/queue", middleware.RequirePermission(perms, "queue:monitor"), deps.NotificationService.Monitor)
	}
}

// healthCheck kini memeriksa DUA database: PostgreSQL dan MongoDB.
// Bila PostgreSQL gagal -> 503 dengan pesan PostgreSQL
// Bila MongoDB gagal -> 503 dengan pesan MongoDB
// Bila keduanya berjalan -> 200 OK
func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("PostgreSQL tidak dapat dihubungi")
		}
		if mongoClient != nil {
			if err := mongoClient.Ping(ctx, nil); err != nil {
				return helper.ServiceUnavailable("MongoDB tidak dapat dihubungi")
			}
		}
		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", nil)
	}
}
