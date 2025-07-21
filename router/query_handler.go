package router

import (
	"context"
	"time"
	"wealth-bond-quote-service/service"

	"github.com/gofiber/fiber/v2"
)

// QueryHandler 查询处理器
type QueryHandler struct {
	queryService *service.BondQueryService
}

// NewQueryHandler 创建查询处理器
func NewQueryHandler(queryService *service.BondQueryService) *QueryHandler {
	return &QueryHandler{
		queryService: queryService,
	}
}

// RegisterRoutes 注册路由
func (h *QueryHandler) RegisterRoutes(app *fiber.App) {
	queryGroup := app.Group("/v1/api/bond/query")

	// 健康检查接口
	queryGroup.Get("/ping", h.Ping)
	// 导出日终数据
	queryGroup.Get("/daily", h.ExportDailyData)
	// 导出历史数据
	queryGroup.Get("/history", h.ExportHistoryData)
}

// ExportDailyData 导出日终数据
func (h *QueryHandler) ExportDailyData(c *fiber.Ctx) error {
	var param service.DateRangeParam
	if err := c.QueryParser(&param); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
	}

	// 参数验证
	if param.StartDate == "" || param.EndDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": 400,
			"msg":  "开始日期和结束日期不能为空",
		})
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// 导出数据
	result, err := h.queryService.ExportDailyData(ctx, param)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": 500,
			"msg":  "导出失败: " + err.Error(),
		})
	}

	// 返回下载链接
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code": 200,
		"msg":  "导出成功",
		"data": result,
	})
}

// ExportHistoryData 导出历史数据
func (h *QueryHandler) ExportHistoryData(c *fiber.Ctx) error {
	var param service.TimeRangeParam
	if err := c.QueryParser(&param); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
	}

	// 参数验证
	if param.StartDate == "" || param.EndDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": 400,
			"msg":  "开始日期和结束日期不能为空",
		})
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// 导出数据
	result, err := h.queryService.ExportHistoryData(ctx, param)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": 500,
			"msg":  "导出失败: " + err.Error(),
		})
	}

	// 返回下载链接
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code": 200,
		"msg":  "导出成功",
		"data": result,
	})
}

// Ping 健康检查接口
func (h *QueryHandler) Ping(c *fiber.Ctx) error {
	// 返回成功响应
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code": 200,
		"msg":  "健康检查成功",
		"data": fiber.Map{
			"service": "wealth-bond-quote-service",
			"status":  "running",
			"time":    time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}
