package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	qc := router.Group("/qc", protected, tenantCtx)

	qc.Get("/stats", h.GetStats)

	params := qc.Group("/parameters")
	params.Get("/", h.ListParameters)
	params.Post("/", h.CreateParameter)
	params.Get("/:id", h.GetParameter)
	params.Put("/:id", h.UpdateParameter)
	params.Delete("/:id", h.DeleteParameter)

	inspections := qc.Group("/inspections")
	inspections.Get("/", h.ListInspections)
	inspections.Post("/", h.CreateInspection)
	inspections.Get("/:id", h.GetInspection)
	inspections.Post("/:id/void", h.VoidInspection)

	hygiene := qc.Group("/hygiene")
	hygiene.Get("/", h.ListHygieneChecks)
	hygiene.Post("/", h.CreateHygieneCheck)
	hygiene.Delete("/:id", h.DeleteHygieneCheck)

	preop := qc.Group("/preop")
	preop.Get("/", h.ListPreOpChecks)
	preop.Post("/", h.CreatePreOpCheck)
	preop.Delete("/:id", h.DeletePreOpCheck)

	packaging := qc.Group("/packaging")
	packaging.Get("/", h.ListPackagingChecks)
	packaging.Post("/", h.CreatePackagingCheck)
	packaging.Get("/:id", h.GetPackagingCheck)
	packaging.Put("/:id", h.UpdatePackagingCheck)
	packaging.Delete("/:id", h.DeletePackagingCheck)

	armada := qc.Group("/armada")
	armada.Get("/", h.ListArmadaChecks)
	armada.Post("/", h.CreateArmadaCheck)
	armada.Get("/:id", h.GetArmadaCheck)
	armada.Put("/:id", h.UpdateArmadaCheck)
	armada.Delete("/:id", h.DeleteArmadaCheck)

	incoming := qc.Group("/incoming")
	incoming.Get("/", h.ListIncomingChecks)
	incoming.Post("/", h.CreateIncomingCheck)
	incoming.Get("/:id", h.GetIncomingCheck)
	incoming.Put("/:id", h.UpdateIncomingCheck)
	incoming.Delete("/:id", h.DeleteIncomingCheck)

	// Sieving Monitoring (MNA, CMT, MNM)
	sieving := qc.Group("/sieving")
	sieving.Get("/mna", h.ListSievingLogs)
	sieving.Post("/mna", h.CreateSievingLog)
	sieving.Get("/mna/:id", h.GetSievingLog)
	sieving.Put("/mna/:id", h.UpdateSievingLog)
	sieving.Delete("/mna/:id", h.DeleteSievingLog)

	sieving.Get("/cmt", h.ListMagnetTrapLogs)
	sieving.Post("/cmt", h.CreateMagnetTrapLog)
	sieving.Get("/cmt/:id", h.GetMagnetTrapLog)
	sieving.Put("/cmt/:id", h.UpdateMagnetTrapLog)
	sieving.Delete("/cmt/:id", h.DeleteMagnetTrapLog)

	sieving.Get("/mnm", h.ListMeshBoltLogs)
	sieving.Post("/mnm", h.CreateMeshBoltLog)
	sieving.Get("/mnm/:id", h.GetMeshBoltLog)
	sieving.Put("/mnm/:id", h.UpdateMeshBoltLog)
	sieving.Delete("/mnm/:id", h.DeleteMeshBoltLog)

	// Oven Monitoring (MNO)
	oven := qc.Group("/oven")
	oven.Get("/mno", h.ListOvenLogs)
	oven.Post("/mno", h.CreateOvenLog)
	oven.Get("/mno/:id", h.GetOvenLog)
	oven.Put("/mno/:id", h.UpdateOvenLog)
	oven.Delete("/mno/:id", h.DeleteOvenLog)

	// Packing Room Quality Controls (MNJ & MNG)
	packing := qc.Group("/packing")
	packing.Get("/mnj", h.ListNeedleLogs)
	packing.Post("/mnj", h.CreateNeedleLog)
	packing.Get("/mnj/:id", h.GetNeedleLog)
	packing.Put("/mnj/:id", h.UpdateNeedleLog)
	packing.Delete("/mnj/:id", h.DeleteNeedleLog)

	packing.Get("/mng", h.ListScissorLogs)
	packing.Post("/mng", h.CreateScissorLog)
	packing.Get("/mng/:id", h.GetScissorLog)
	packing.Put("/mng/:id", h.UpdateScissorLog)
	packing.Delete("/mng/:id", h.DeleteScissorLog)

	// Metal Detector CCP (CPP & VMD)
	metal := qc.Group("/metal-detector")
	metal.Get("/cpp", h.ListMetalCleaningLogs)
	metal.Post("/cpp", h.CreateMetalCleaningLog)
	metal.Get("/cpp/:id", h.GetMetalCleaningLog)
	metal.Put("/cpp/:id", h.UpdateMetalCleaningLog)
	metal.Delete("/cpp/:id", h.DeleteMetalCleaningLog)

	metal.Get("/vmd", h.ListMetalVerificationLogs)
	metal.Post("/vmd", h.CreateMetalVerificationLog)
	metal.Get("/vmd/:id", h.GetMetalVerificationLog)
	metal.Put("/vmd/:id", h.UpdateMetalVerificationLog)
	metal.Delete("/vmd/:id", h.DeleteMetalVerificationLog)

	// Finished Goods Quality Inspection (PJD - Form QC-004-PJD.05)
	finishedGoods := qc.Group("/finished-goods")
	finishedGoods.Get("/pjd", h.ListFinishedGoodsLogs)
	finishedGoods.Post("/pjd", h.CreateFinishedGoodsLog)
	finishedGoods.Get("/pjd/:id", h.GetFinishedGoodsLog)
	finishedGoods.Put("/pjd/:id", h.UpdateFinishedGoodsLog)
	finishedGoods.Delete("/pjd/:id", h.DeleteFinishedGoodsLog)

	// Packaging Label Inspection (PLK - Form QC-018-PLK)
	labelInspection := qc.Group("/label-inspection")
	labelInspection.Get("/plk", h.ListPackagingLabelChecks)
	labelInspection.Post("/plk", h.CreatePackagingLabelCheck)
	labelInspection.Get("/plk/:id", h.GetPackagingLabelCheck)
	labelInspection.Put("/plk/:id", h.UpdatePackagingLabelCheck)
	labelInspection.Delete("/plk/:id", h.DeletePackagingLabelCheck)

	// Container Checklist (CCO - Form QC-008-CCO.02)
	container := qc.Group("/container")
	container.Get("/cco", h.ListContainerChecklists)
	container.Post("/cco", h.CreateContainerChecklist)
	container.Get("/cco/:id", h.GetContainerChecklist)
	container.Put("/cco/:id", h.UpdateContainerChecklist)
	container.Delete("/cco/:id", h.DeleteContainerChecklist)

	// Lot traceability (moved from legacy certification module)
	router.Get("/traceability/lot/:lot", protected, tenantCtx, h.TraceLot)
}
