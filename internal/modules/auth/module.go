package auth

import (
	"log/slog"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/notify"
	"github.com/divinecoid/one-backend/internal/modules/auth/application"
	"github.com/divinecoid/one-backend/internal/modules/auth/delivery/http"
	authDomain "github.com/divinecoid/one-backend/internal/modules/auth/domain"
	"github.com/divinecoid/one-backend/internal/modules/auth/infrastructure"
	companyDomain "github.com/divinecoid/one-backend/internal/modules/company/domain"
	tenantDomain "github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	UseCase application.AuthUseCase
	Handler *http.Handler
}

// NewModule wires up the auth module and registers HTTP routes
func NewModule(
	db *gorm.DB,
	userRepo userDomain.UserRepository,
	companyRepo companyDomain.CompanyRepository,
	membershipRepo tenantDomain.CompanyMembershipRepository,
	notifier notify.Notifier,
	frontendBaseURL string,
	router fiber.Router,
	jwtSecret string,
	jwtExpiry time.Duration,
) *Module {
	if err := db.AutoMigrate(&authDomain.PasswordResetToken{}); err != nil {
		slog.Error("failed to migrate password reset token schema", "error", err)
	}

	resetTokenRepo := infrastructure.NewPasswordResetTokenRepository(db)

	uc := application.NewAuthUseCase(userRepo, companyRepo, membershipRepo, resetTokenRepo, notifier, frontendBaseURL, jwtSecret, jwtExpiry)
	handler := http.NewHandler(uc)

	handler.RegisterRoutes(router, jwtSecret)

	slog.Info("auth module initialized")

	return &Module{
		UseCase: uc,
		Handler: handler,
	}
}
