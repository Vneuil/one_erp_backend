package domain

import (
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// PackagingLabelCheckLog represents PLK (Form Pemeriksaan Label Kemasan QC — QC-018-PLK)
type PackagingLabelCheckLog struct {
	types.BaseEntity
	DocumentNo            string                    `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	CheckDate             string                    `gorm:"type:varchar(10);not null;index" json:"checkDate"`
	CheckMonth            string                    `gorm:"type:varchar(7);not null;index" json:"checkMonth"`
	HandoverReportID      *uuid.UUID                `gorm:"type:uuid;index" json:"handoverReportId,omitempty"`
	HandoverReportNo      string                    `gorm:"type:varchar(50)" json:"handoverReportNo,omitempty"`
	BatchNo               string                    `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	FairtradeStatus       string                    `gorm:"type:varchar(50);not null;default:'NON FAIRTRADE'" json:"fairtradeStatus"`
	Certification         string                    `gorm:"type:varchar(100);default:'NATURAL'" json:"certification"`
	QCInspectorName       string                    `gorm:"type:varchar(150);not null" json:"qcInspectorName"`
	QCInspectorSignDate   *string                   `gorm:"type:varchar(20)" json:"qcInspectorSignDate,omitempty"`
	QCSupervisorName      string                    `gorm:"type:varchar(150)" json:"qcSupervisorName,omitempty"`
	QCSupervisorSignDate  *string                   `gorm:"type:varchar(20)" json:"qcSupervisorSignDate,omitempty"`
	OverallDecision       string                    `gorm:"type:varchar(20);not null;default:'lolos'" json:"overallDecision"` // lolos, ditolak
	Notes                 string                    `gorm:"type:text" json:"notes,omitempty"`
	Items                 []PackagingLabelCheckItem `gorm:"foreignKey:LogID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

func (PackagingLabelCheckLog) TableName() string { return "qc_packaging_label_checks" }

// PackagingLabelCheckItem represents one product item line inspected for labels
type PackagingLabelCheckItem struct {
	types.BaseEntity
	LogID              uuid.UUID `gorm:"type:uuid;not null;index" json:"logId"`
	ItemNo             int       `gorm:"not null" json:"itemNo"`
	ItemName           string    `gorm:"type:varchar(255);not null" json:"itemName"`
	ItemCode           string    `gorm:"type:varchar(100);not null" json:"itemCode"`
	BatchNo            string    `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	BuyerName          string    `gorm:"type:varchar(255)" json:"buyerName,omitempty"`
	PackagingType      string    `gorm:"type:varchar(100)" json:"packagingType,omitempty"`
	StickerNo          string    `gorm:"type:varchar(100)" json:"stickerNo,omitempty"`
	IsBatchMatch       bool      `gorm:"not null;default:true" json:"isBatchMatch"`
	IsExpiryMatch      bool      `gorm:"not null;default:true" json:"isExpiryMatch"`
	IsCertLogoMatch    bool      `gorm:"not null;default:true" json:"isCertLogoMatch"`
	IsPrintQualityGood bool      `gorm:"not null;default:true" json:"isPrintQualityGood"`
	IsPositionAligned  bool      `gorm:"not null;default:true" json:"isPositionAligned"`
	InspectedQty       int       `gorm:"not null;default:0" json:"inspectedQty"`
	PassedQty          int       `gorm:"not null;default:0" json:"passedQty"`
	RejectedQty        int       `gorm:"not null;default:0" json:"rejectedQty"`
	ItemDecision       string    `gorm:"type:varchar(20);not null;default:'lolos'" json:"itemDecision"` // lolos, ditolak
	CorrectiveAction   string    `gorm:"type:varchar(255)" json:"correctiveAction,omitempty"`
	Notes              string    `gorm:"type:varchar(255)" json:"notes,omitempty"`
}

func (PackagingLabelCheckItem) TableName() string { return "qc_packaging_label_check_items" }

// ContainerChecklistLog represents CCO (Checklist Container QC — QC-008-CCO.02)
type ContainerChecklistLog struct {
	types.BaseEntity
	DocumentNo            string  `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	CheckDate             string  `gorm:"type:varchar(10);not null;index" json:"checkDate"`
	CheckMonth            string  `gorm:"type:varchar(7);not null;index" json:"checkMonth"`

	// Driver Identity
	IDCardType            string  `gorm:"type:varchar(20);not null;default:'KTP'" json:"idCardType"` // KTP or SIM
	IDCardNumber          string  `gorm:"type:varchar(100);not null" json:"idCardNumber"`
	DriverName            string  `gorm:"type:varchar(150);not null" json:"driverName"`
	BirthDetails          string  `gorm:"type:varchar(150)" json:"birthDetails,omitempty"` // Tempat Tanggal Lahir (TTL)
	Gender                string  `gorm:"type:varchar(20)" json:"gender,omitempty"`
	Address               string  `gorm:"type:text" json:"address,omitempty"`
	Religion              string  `gorm:"type:varchar(50)" json:"religion,omitempty"`
	MaritalStatus         string  `gorm:"type:varchar(50)" json:"maritalStatus,omitempty"`
	CardValidity          string  `gorm:"type:varchar(50)" json:"cardValidity,omitempty"`
	TransporterCompany    string  `gorm:"type:varchar(150);not null" json:"transporterCompany"`
	VehicleLicensePlate   string  `gorm:"type:varchar(50);not null" json:"vehicleLicensePlate"` // Nopol

	// Container Identity
	ContainerName         string  `gorm:"type:varchar(100);not null" json:"containerName"`
	ContainerNumber       string  `gorm:"type:varchar(100);not null;index" json:"containerNumber"`
	ContainerSize         string  `gorm:"type:varchar(50);not null;default:'20ft'" json:"containerSize"` // 20ft, 40ft, 40ft HC
	SealNumber            string  `gorm:"type:varchar(100);not null" json:"sealNumber"` // Nomor seal / CAP
	ShippingInstructionNo string  `gorm:"type:varchar(100)" json:"shippingInstructionNo,omitempty"`
	EIRNumber             string  `gorm:"type:varchar(100)" json:"eirNumber,omitempty"`
	ERNumber              string  `gorm:"type:varchar(100)" json:"erNumber,omitempty"`

	// Suspicious Items Check (Barang / Bahan Mencurigakan)
	CabinAreaStatus       string  `gorm:"type:varchar(50);not null;default:'aman'" json:"cabinAreaStatus"`       // aman, mencurigakan
	UnderVehicleStatus    string  `gorm:"type:varchar(50);not null;default:'aman'" json:"underVehicleStatus"`    // aman, mencurigakan
	InsideContainerStatus string  `gorm:"type:varchar(50);not null;default:'aman'" json:"insideContainerStatus"` // aman, mencurigakan
	SuspiciousNotes       string  `gorm:"type:text" json:"suspiciousNotes,omitempty"`

	// Cleanliness Check (Kebersihan Ruang Dalam Kontainer - Bebas cairan/padatan/noda)
	FloorCleanliness      string  `gorm:"type:varchar(50);not null;default:'bersih'" json:"floorCleanliness"`      // bersih, kotor
	RightWallCleanliness  string  `gorm:"type:varchar(50);not null;default:'bersih'" json:"rightWallCleanliness"`  // bersih, kotor
	LeftWallCleanliness   string  `gorm:"type:varchar(50);not null;default:'bersih'" json:"leftWallCleanliness"`   // bersih, kotor
	RoofCleanliness       string  `gorm:"type:varchar(50);not null;default:'bersih'" json:"roofCleanliness"`       // bersih, kotor
	DoorCleanliness       string  `gorm:"type:varchar(50);not null;default:'bersih'" json:"doorCleanliness"`       // bersih, kotor
	CleanlinessNotes      string  `gorm:"type:text" json:"cleanlinessNotes,omitempty"`

	// Leak / Hole Check (Posisi Dua Pintu Tertutup Rapat - Bebas bocor cahaya/air)
	FloorHoleStatus       string  `gorm:"type:varchar(50);not null;default:'rapat'" json:"floorHoleStatus"`      // rapat, bocor
	RightWallHoleStatus   string  `gorm:"type:varchar(50);not null;default:'rapat'" json:"rightWallHoleStatus"`  // rapat, bocor
	LeftWallHoleStatus    string  `gorm:"type:varchar(50);not null;default:'rapat'" json:"leftWallHoleStatus"`   // rapat, bocor
	RoofHoleStatus        string  `gorm:"type:varchar(50);not null;default:'rapat'" json:"roofHoleStatus"`       // rapat, bocor
	DoorHoleStatus        string  `gorm:"type:varchar(50);not null;default:'rapat'" json:"doorHoleStatus"`       // rapat, bocor
	HoleNotes             string  `gorm:"type:text" json:"holeNotes,omitempty"`

	// Timestamps & Pallet
	SealTime              string  `gorm:"type:varchar(20)" json:"sealTime,omitempty"`      // Jam Kontainer disegel
	CapTime               string  `gorm:"type:varchar(20)" json:"capTime,omitempty"`       // Jam Kontainer dipasang CAP
	DepartureTime         string  `gorm:"type:varchar(20)" json:"departureTime,omitempty"` // Jam Berangkat
	PalletDetails         string  `gorm:"type:varchar(150)" json:"palletDetails,omitempty"`
	ShipmentDestination   string  `gorm:"type:varchar(255);not null" json:"shipmentDestination"` // Kiriman Ke

	// Verification & Approval
	InspectorQC           string  `gorm:"type:varchar(150);not null" json:"inspectorQc"` // Dibuat Oleh QC
	QCSignDate            *string `gorm:"type:varchar(20)" json:"qcSignDate,omitempty"`
	VerifierQA            string  `gorm:"type:varchar(150);not null" json:"verifierQa"` // Diverifikasi Oleh QA
	QASignDate            *string `gorm:"type:varchar(20)" json:"qaSignDate,omitempty"`
	OverallDecision       string  `gorm:"type:varchar(20);not null;default:'lolos'" json:"overallDecision"` // lolos, ditolak
	Notes                 string  `gorm:"type:text" json:"notes,omitempty"`
}

func (ContainerChecklistLog) TableName() string { return "qc_container_checklists" }
