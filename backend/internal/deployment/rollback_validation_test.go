package deployment

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestIsCurrentDeploymentID(t *testing.T) {
	current := uuid.MustParse("741151a3-ee89-4357-a797-289b50be2431")
	other := uuid.MustParse("9329ad75-b680-4adb-97b5-c7087cd1ae37")

	if !isCurrentDeploymentID(pgtype.UUID{Bytes: current, Valid: true}, current) {
		t.Fatal("matching active deployment should be current")
	}
	if isCurrentDeploymentID(pgtype.UUID{Bytes: current, Valid: true}, other) {
		t.Fatal("different deployment must not be current")
	}
	if isCurrentDeploymentID(pgtype.UUID{}, current) {
		t.Fatal("invalid active deployment id must not match")
	}
}
