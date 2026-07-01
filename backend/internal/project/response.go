package project

import (
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
)

type Response struct {
	ID                 string  `json:"id"`
	UserID             string  `json:"userId"`
	Name               string  `json:"name"`
	RepoURL            string  `json:"repoUrl"`
	Branch             string  `json:"branch"`
	Subdomain          string  `json:"subdomain"`
	DeployMode         string  `json:"deployMode"`
	MainService        *string `json:"mainService"`
	AppPort            int32   `json:"appPort"`
	AllocatedPort      *int32  `json:"allocatedPort"`
	MemoryLimitMb      int32   `json:"memoryLimitMb"`
	CPULimit           float64 `json:"cpuLimit"`
	Status             string  `json:"status"`
	ActiveDeploymentID *string `json:"activeDeploymentId"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

func ResponseFromDB(project db.Project) Response {
	return Response{
		ID:                 project.ID.String(),
		UserID:             project.UserID.String(),
		Name:               project.Name,
		RepoURL:            project.RepoUrl,
		Branch:             project.Branch,
		Subdomain:          project.Subdomain,
		DeployMode:         project.DeployMode,
		MainService:        project.MainService,
		AppPort:            project.AppPort,
		AllocatedPort:      project.AllocatedPort,
		MemoryLimitMb:      project.MemoryLimitMb,
		CPULimit:           numericToResponseFloat(project.CpuLimit),
		Status:             project.Status,
		ActiveDeploymentID: uuidString(project.ActiveDeploymentID),
		CreatedAt:          formatTimestamp(project.CreatedAt.Time, project.CreatedAt.Valid),
		UpdatedAt:          formatTimestamp(project.UpdatedAt.Time, project.UpdatedAt.Valid),
	}
}

func ResponsesFromDB(projects []db.Project) []Response {
	out := make([]Response, 0, len(projects))
	for _, item := range projects {
		out = append(out, ResponseFromDB(item))
	}
	return out
}

func numericToResponseFloat(value pgtype.Numeric) float64 {
	if !value.Valid || value.Int == nil {
		return 0
	}
	f, _ := new(big.Rat).SetFrac(value.Int, big.NewInt(1)).Float64()
	return f * math.Pow10(int(value.Exp))
}

func uuidString(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	formatted := id.String()
	return &formatted
}

func formatTimestamp(t time.Time, valid bool) string {
	if !valid {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
