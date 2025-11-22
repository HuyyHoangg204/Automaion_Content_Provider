package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/onegreenvn/green-provider-services-backend/internal/database/repository"
	"github.com/onegreenvn/green-provider-services-backend/internal/models"
	"github.com/onegreenvn/green-provider-services-backend/internal/services"
	"gorm.io/gorm"
)

type MachineHandler struct {
	machineService *services.MachineService
}

func NewMachineHandler(db *gorm.DB) *MachineHandler {
	boxRepo := repository.NewBoxRepository(db)
	appRepo := repository.NewAppRepository(db)
	userRepo := repository.NewUserRepository(db)

	return &MachineHandler{
		machineService: services.NewMachineService(boxRepo, appRepo, userRepo),
	}
}

// RegisterMachine godoc
// @Summary Register a machine
// @Description Register a new machine or return existing machine info. This is a public endpoint for machines to self-register.
// @Tags machines
// @Accept json
// @Produce json
// @Param request body models.RegisterMachineRequest true "Machine registration request"
// @Success 200 {object} models.RegisterMachineResponse
// @Success 201 {object} models.RegisterMachineResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/machines/register [post]
func (h *MachineHandler) RegisterMachine(c *gin.Context) {
	var req models.RegisterMachineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.machineService.RegisterMachine(req.MachineID, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "no users found") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register machine", "details": err.Error()})
		return
	}

	// Return 201 if new machine, 200 if existing
	statusCode := http.StatusCreated
	if response.Message == "Machine already registered" {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, response)
}

// GetFrpConfigByMachineID godoc
// @Summary Get FRP configuration for a machine
// @Description Get FRP configuration and subdomain for a specific machine by machine_id
// @Tags machines
// @Accept json
// @Produce json
// @Param machine_id path string true "Machine ID"
// @Success 200 {object} models.RegisterAppResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/machines/{machine_id}/frp-config [get]
func (h *MachineHandler) GetFrpConfigByMachineID(c *gin.Context) {
	machineID := c.Param("machine_id")
	if machineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "machine_id is required"})
		return
	}

	response, err := h.machineService.GetFrpConfigByMachineID(machineID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Machine not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get FRP config", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTunnelURLByMachineID godoc
// @Summary Update tunnel URL for a machine
// @Description Update tunnel URL for the machine's Automation app
// @Tags machines
// @Accept json
// @Produce json
// @Param machine_id path string true "Machine ID"
// @Param request body models.UpdateTunnelURLRequest true "Tunnel URL update request"
// @Success 200 {object} models.UpdateTunnelURLResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/machines/{machine_id}/tunnel-url [put]
func (h *MachineHandler) UpdateTunnelURLByMachineID(c *gin.Context) {
	machineID := c.Param("machine_id")
	if machineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "machine_id is required"})
		return
	}

	var req models.UpdateTunnelURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.machineService.UpdateTunnelURLByMachineID(machineID, req.TunnelURL)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Machine not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tunnel URL", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SendHeartbeat godoc
// @Summary Send machine heartbeat
// @Description Send heartbeat to update machine status and last seen time
// @Tags machines
// @Accept json
// @Produce json
// @Param machine_id path string true "Machine ID"
// @Param request body models.HeartbeatRequest true "Heartbeat request"
// @Success 200 {object} models.HeartbeatResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/machines/{machine_id}/heartbeat [post]
func (h *MachineHandler) SendHeartbeat(c *gin.Context) {
	machineID := c.Param("machine_id")
	if machineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "machine_id is required"})
		return
	}

	var req models.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	response, err := h.machineService.SendHeartbeat(machineID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Machine not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process heartbeat", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

