package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/blueship581/port-mooring-window-safety/backend/internal/config"
	"github.com/blueship581/port-mooring-window-safety/backend/internal/model"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, cfg config.Config, log *slog.Logger) (*gorm.DB, *redis.Client, error) {
	var dialector gorm.Dialector
	switch cfg.DatabaseDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseDSN)
	case "mysql":
		dialector = mysql.Open(cfg.DatabaseDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DatabaseDSN)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver %q", cfg.DatabaseDriver)
	}
	logLevel := logger.Warn
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 20; attempt++ {
		db, err = gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logLevel)})
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil && sqlDB.PingContext(ctx) == nil {
				break
			}
			if dbErr != nil {
				err = dbErr
			} else {
				err = sqlDB.PingContext(ctx)
			}
		}
		log.Warn("database not ready", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("connect database: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, nil, err
	}
	if err := Seed(ctx, db); err != nil {
		return nil, nil, err
	}
	var redisClient *redis.Client
	if cfg.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, nil, fmt.Errorf("connect redis: %w", err)
		}
	}
	return db, redisClient, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{}, &model.AuditLog{},
		&model.VesselCall{},
		&model.MooringPlan{},
		&model.WeatherWindow{},
		&model.SafetyClearance{},
		&model.RopeInspection{},
	)
}

func Seed(ctx context.Context, db *gorm.DB) error {
	var users int64
	if err := db.WithContext(ctx).Model(&model.User{}).Count(&users).Error; err != nil {
		return err
	}
	if users == 0 {
		password, err := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		seedUsers := []model.User{
			{Username: "admin", DisplayName: "系统管理员", PasswordHash: string(password), Role: model.RoleAdmin, Active: true},
			{Username: "reviewer", DisplayName: "质量复核员", PasswordHash: string(password), Role: model.RoleReviewer, Active: true},
			{Username: "operator", DisplayName: "现场操作员", PasswordHash: string(password), Role: model.RoleOperator, Active: true},
			{Username: "viewer", DisplayName: "只读观察员", PasswordHash: string(password), Role: model.RoleViewer, Active: true},
		}
		if err := db.WithContext(ctx).Create(&seedUsers).Error; err != nil {
			return err
		}
	}

	if err := seedVesselCall(ctx, db); err != nil {
		return err
	}

	if err := seedMooringPlan(ctx, db); err != nil {
		return err
	}

	if err := seedWeatherWindow(ctx, db); err != nil {
		return err
	}

	if err := seedRopeInspection(ctx, db); err != nil {
		return err
	}

	if err := seedSafetyClearance(ctx, db); err != nil {
		return err
	}

	return nil
}

func seedVesselCall(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.VesselCall{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.VesselCall{

		{BaseModel: model.BaseModel{Code: "VC-001", Name: "船舶靠泊示例一", Status: "planned", Version: 1,
			Description: "用于启动验证和主要流程演示的船舶靠泊记录"}, Facility: "港口系泊安全窗口评估区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-01"},

		{BaseModel: model.BaseModel{Code: "VC-002", Name: "船舶靠泊示例二", Status: "approach", Version: 1,
			Description: "用于启动验证和主要流程演示的船舶靠泊记录"}, Facility: "港口系泊安全窗口评估区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-02"},

		{BaseModel: model.BaseModel{Code: "VC-003", Name: "船舶靠泊示例三", Status: "moored", Version: 1,
			Description: "用于启动验证和主要流程演示的船舶靠泊记录"}, Facility: "港口系泊安全窗口评估区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedMooringPlan(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.MooringPlan{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.MooringPlan{

		{BaseModel: model.BaseModel{Code: "MP-001", Name: "系泊方案示例一", Status: "draft", Version: 1,
			Description: "用于启动验证和主要流程演示的系泊方案记录"}, Facility: "港口系泊安全窗口评估区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-01"},

		{BaseModel: model.BaseModel{Code: "MP-002", Name: "系泊方案示例二", Status: "review", Version: 1,
			Description: "用于启动验证和主要流程演示的系泊方案记录"}, Facility: "港口系泊安全窗口评估区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-02"},

		{BaseModel: model.BaseModel{Code: "MP-003", Name: "系泊方案示例三", Status: "approved", Version: 1,
			Description: "用于启动验证和主要流程演示的系泊方案记录"}, Facility: "港口系泊安全窗口评估区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedWeatherWindow(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.WeatherWindow{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.WeatherWindow{

		{BaseModel: model.BaseModel{Code: "WW-001", Name: "风浪窗口示例一", Status: "forecast", Version: 1,
			Description: "用于启动验证和主要流程演示的风浪窗口记录"}, Facility: "港口系泊安全窗口评估区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-01"},

		{BaseModel: model.BaseModel{Code: "WW-002", Name: "风浪窗口示例二", Status: "safe", Version: 1,
			Description: "用于启动验证和主要流程演示的风浪窗口记录"}, Facility: "港口系泊安全窗口评估区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-02"},

		{BaseModel: model.BaseModel{Code: "WW-003", Name: "风浪窗口示例三", Status: "restricted", Version: 1,
			Description: "用于启动验证和主要流程演示的风浪窗口记录"}, Facility: "港口系泊安全窗口评估区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-510-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedRopeInspection(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.RopeInspection{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.RopeInspection{

		{BaseModel: model.BaseModel{Code: "RI-001", Name: "MP-001 艏缆-1 缆绳检查", Status: "closed", Version: 1,
			Description: "靠泊前例行检查，缆绳状态良好"}, PlanCode: "MP-001", LinePosition: "艏缆-1",
			InspectedAt: now.Add(-2 * time.Hour), Inspector: "operator", DefectLevel: "none", Conclusion: "passed"},

		{BaseModel: model.BaseModel{Code: "RI-002", Name: "MP-002 艉缆-2 缆绳检查", Status: "open", Version: 1,
			Description: "现场发现断股，需换绳后复查"}, PlanCode: "MP-002", LinePosition: "艉缆-2",
			InspectedAt: now.Add(-50 * time.Minute), Inspector: "operator", DefectLevel: "broken_strand",
			Conclusion: "replace_pending", BlockedReason: "艉缆-2 断股，处置结论：待换绳"},

		{BaseModel: model.BaseModel{Code: "RI-003", Name: "MP-003 横缆-1 缆绳检查", Status: "closed", Version: 1,
			Description: "轻微磨损，观察使用"}, PlanCode: "MP-003", LinePosition: "横缆-1",
			InspectedAt: now.Add(-30 * time.Minute), Inspector: "reviewer", DefectLevel: "wear_minor", Conclusion: "monitor"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedSafetyClearance(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.SafetyClearance{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	var blockedInspection model.RopeInspection
	blockedInspectionID := uint(0)
	if err := db.WithContext(ctx).Where("code = ?", "RI-002").First(&blockedInspection).Error; err == nil {
		blockedInspectionID = blockedInspection.ID
	}
	items := []model.SafetyClearance{

		{BaseModel: model.BaseModel{Code: "SC-001", Name: "安全许可示例一", Status: "pending", Version: 1,
			Description: "用于启动验证和主要流程演示的安全许可记录"}, Facility: "港口系泊安全窗口评估区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "WW-001",
			WindowVersion: 1, PlanCode: "MP-001", SubmittedBy: "operator", SubmittedAt: timePointer(now.Add(-15 * time.Minute))},

		{BaseModel: model.BaseModel{Code: "SC-002", Name: "安全许可示例二", Status: "cleared", Version: 1,
			Description: "用于启动验证和主要流程演示的安全许可记录"}, Facility: "港口系泊安全窗口评估区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "WW-002",
			WindowVersion: 1, PlanCode: "MP-003", SubmittedBy: "operator", SubmittedAt: timePointer(now.Add(-45 * time.Minute)), ConfirmedBy: "reviewer", ConfirmedAt: timePointer(now.Add(-30 * time.Minute))},

		{BaseModel: model.BaseModel{Code: "SC-003", Name: "安全许可示例三", Status: "restricted", Version: 1,
			Description: "用于启动验证和主要流程演示的安全许可记录"}, Facility: "港口系泊安全窗口评估区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "WW-003", WindowVersion: 1},

		{BaseModel: model.BaseModel{Code: "SC-004", Name: "安全许可示例四", Status: "pending", Version: 1,
			Description: "缆绳检查发现断股，待复核许可已退回，换绳复查通过后可重新提交"}, Facility: "港口系泊安全窗口评估区域2", Owner: "运行一组",
			Category: "重点", RiskLevel: "high", MetricValue: 31.0, MetricUnit: "score",
			EffectiveAt: now.Add(2 * time.Hour), Evidence: "等待换绳复查", RelatedCode: "WW-002",
			WindowVersion: 1, PlanCode: "MP-002",
			BlockedReason: "检查 RI-002：艉缆-2 断股，处置结论：待换绳", BlockedInspectionID: blockedInspectionID},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func timePointer(value time.Time) *time.Time { return &value }
