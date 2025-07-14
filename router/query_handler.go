package router

import (
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

	// 导出当前最新行情
	queryGroup.Get("/current-latest", h.ExportCurrentLatestQuotes)
	// 导出日终数据
	queryGroup.Get("/daily-end", h.ExportDailyEndData)
	// 导出时间段数据
	queryGroup.Get("/time-range", h.ExportTimeRangeData)
}

// ExportCurrentLatestQuotes 导出当前最新行情
func (h *QueryHandler) ExportCurrentLatestQuotes(c *fiber.Ctx) error {
	// 导出数据
	filename, err := h.queryService.ExportCurrentLatestQuotes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": 500,
			"msg":  "导出失败: " + err.Error(),
		})
	}

	// 返回文件路径
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code": 200,
		"msg":  "导出成功",
		"data": fiber.Map{
			"filename": filename,
		},
	})
}

// ExportDailyEndData 导出日终数据
func (h *QueryHandler) ExportDailyEndData(c *fiber.Ctx) error {
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

	// 导出数据
	filename, err := h.queryService.ExportDailyEndData(param)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": 500,
			"msg":  "导出失败: " + err.Error(),
		})
	}

	// 返回文件路径
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code": 200,
		"msg":  "导出成功",
		"data": fiber.Map{
			"filename": filename,
		},
	})
}

// ExportTimeRangeData 导出时间段数据
func (h *QueryHandler) ExportTimeRangeData(c *fiber.Ctx) error {
	var param service.TimeRangeParam
	if err := c.QueryParser(&param); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
	}

	// 参数验证
	if param.Date == "" || param.StartTime == "" || param.EndTime == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": 400,
			"msg":  "日期、开始时间和结束时间不能为空",
		})
	}

	// 导出数据
	filename, err := h.queryService.ExportTimeRangeData(param)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": 500,
			"msg":  "导出失败: " + err.Error(),
		})
	}

	// 返回文件路径
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code": 200,
		"msg":  "导出成功",
		"data": fiber.Map{
			"filename": filename,
		},
	})
}
