package consoleapp

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trembon/switch-library-manager/backend/db"
	"github.com/trembon/switch-library-manager/backend/process"
	"github.com/trembon/switch-library-manager/backend/settings"
)

func TestOrganizationEnabledForMoveOnlyOption(t *testing.T) {
	if organizationEnabled(settings.OrganizationSettings{MoveScanFilesToLibrary: true}) != true {
		t.Fatal("move-only organization should be enabled")
	}
	if organizationEnabled(settings.OrganizationSettings{}) {
		t.Fatal("organization should stay disabled when all organization options are off")
	}
}

func TestCsvExportFilename(t *testing.T) {
	date := time.Date(2026, time.September, 21, 23, 45, 0, 0, time.FixedZone("test", 2*60*60))
	tests := map[string]string{
		csvGamesKind:          "slm_games.2026-09-21.csv",
		csvMissingGamesKind:   "slm_missing_games.2026-09-21.csv",
		csvMissingUpdatesKind: "slm_missing_updates.2026-09-21.csv",
		csvMissingDLCKind:     "slm_missing_dlc.2026-09-21.csv",
		csvIssuesKind:         "slm_issues.2026-09-21.csv",
	}

	for kind, want := range tests {
		t.Run(kind, func(t *testing.T) {
			if got := csvExportFilename(kind, date); got != want {
				t.Fatalf("csvExportFilename(%q) = %q, want %q", kind, got, want)
			}
		})
	}
}

func TestProcessIssuesExportsOrganizationConflicts(t *testing.T) {
	output := filepath.Join(t.TempDir(), "issues.csv")
	conflicts := []process.OrganizationConflict{{
		Source:      `C:\library\source.nsp`,
		Destination: `C:\library\Game.nsp`,
		OtherSource: `C:\library\Game.nsp`,
		Reason:      "destination already exists",
	}}
	c := CreateConsole("", nil, nil)
	c.processIssues(&db.LocalSwitchFilesDB{Skipped: map[db.ExtendedFileInfo]db.SkippedFile{}}, output, conflicts)

	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[1][2] != "organization-conflict" || !strings.Contains(records[1][1], `C:\library\Game.nsp`) {
		t.Fatalf("issue export = %#v, want organization conflict with destination", records)
	}
}

func TestFlattenMissingDLC(t *testing.T) {
	titles := map[string]process.IncompleteTitle{
		"game-z": {
			Attributes: db.TitleAttributes{Id: "game-z-id", Name: "Zulu"},
			MissingDLC: []process.MissingDLC{
				{Id: "dlc-z-id", Name: "Expansion"},
				{Id: "dlc-a-id", Name: "Adventure"},
				{Id: "dlc-y-id", Name: "Expansion"},
			},
		},
		"game-a": {
			Attributes: db.TitleAttributes{Id: "game-a-id", Name: "Alpha"},
			MissingDLC: []process.MissingDLC{{Id: "dlc-b-id", Name: "Bonus"}},
		},
	}

	rows := flattenMissingDLC(titles)
	if len(rows) != 4 {
		t.Fatalf("flattened missing DLC rows = %d, want 4", len(rows))
	}
	want := []struct{ game, gameID, dlc, dlcID string }{
		{"Alpha", "game-a-id", "Bonus", "dlc-b-id"},
		{"Zulu", "game-z-id", "Adventure", "dlc-a-id"},
		{"Zulu", "game-z-id", "Expansion", "dlc-y-id"},
		{"Zulu", "game-z-id", "Expansion", "dlc-z-id"},
	}
	for i, row := range rows {
		if row.Game.Name != want[i].game || row.Game.Id != want[i].gameID || row.DLC.Name != want[i].dlc || row.DLC.Id != want[i].dlcID {
			t.Errorf("row %d = (%q, %q, %q, %q), want (%q, %q, %q, %q)", i, row.Game.Name, row.Game.Id, row.DLC.Name, row.DLC.Id, want[i].game, want[i].gameID, want[i].dlc, want[i].dlcID)
		}
		csvRow := missingDLCCSVRow(row)
		if len(csvRow) != 3 || csvRow[0] != want[i].game || csvRow[1] != want[i].gameID || csvRow[2] != want[i].dlc+" ["+want[i].dlcID+"]" {
			t.Errorf("CSV row %d = %#v", i, csvRow)
		}
	}
}

func TestCreateCsvFileWritesUTF8BOM(t *testing.T) {
	if got := CreateCsvFile("", []string{"Title"}); got != nil {
		t.Fatalf("CreateCsvFile(\"\") = %#v, want nil", got)
	}

	outputPath := filepath.Join(t.TempDir(), "test_export.csv")
	header := []string{"Title", "TitleId", "Notes"}
	csvFile := CreateCsvFile(outputPath, header)
	if csvFile == nil {
		t.Fatalf("CreateCsvFile(%q) returned nil", outputPath)
	}

	csvFile.Write([]string{"ゼルダの伝説 ティアーズ オブ ザ キングダム", "0100f2c0115b6000", "Japanese characters"})
	csvFile.Write([]string{"Pokémon Legends: Arceus", "01001f5010dfa000", "Accented characters"})
	csvFile.Close()

	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read exported csv file: %v", err)
	}

	expectedBOM := []byte{0xEF, 0xBB, 0xBF}
	if len(raw) < 3 || !bytes.Equal(raw[:3], expectedBOM) {
		t.Fatalf("expected UTF-8 BOM at start of file, got %v", raw[:min(len(raw), 3)])
	}

	content := string(raw[3:])
	if !strings.Contains(content, "ゼルダの伝説") || !strings.Contains(content, "Pokémon") {
		t.Fatalf("exported CSV content missing expected non-ASCII strings: %s", content)
	}
}
