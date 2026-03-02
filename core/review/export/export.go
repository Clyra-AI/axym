package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/review"
)

func JSON(pack review.Pack) ([]byte, error) {
	raw, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func CSV(pack review.Pack) ([]byte, error) {
	var out bytes.Buffer
	writer := csv.NewWriter(&out)

	rows := [][]string{
		{"date", pack.Date},
		{"empty", boolToString(pack.Empty)},
		{"record_count", fmt.Sprintf("%d", pack.RecordCount)},
		{},
		{"exception_class", "count"},
	}
	for _, item := range pack.Exceptions {
		rows = append(rows, []string{item.Name, fmt.Sprintf("%d", item.Count)})
	}
	rows = append(rows, []string{}, []string{"grade", "count"})
	for _, item := range pack.GradeDistribution {
		rows = append(rows, []string{item.Name, fmt.Sprintf("%d", item.Count)})
	}
	rows = append(rows, []string{}, []string{"replay_tier", "count"})
	for _, item := range pack.ReplayTierDistribution {
		rows = append(rows, []string{item.Name, fmt.Sprintf("%d", item.Count)})
	}
	rows = append(rows, []string{}, []string{"attach_status", "count"})
	for _, item := range pack.AttachStatus {
		rows = append(rows, []string{item.Name, fmt.Sprintf("%d", item.Count)})
	}
	rows = append(rows, []string{}, []string{"attach_sla", "count"})
	for _, item := range pack.AttachSLA {
		rows = append(rows, []string{item.Name, fmt.Sprintf("%d", item.Count)})
	}
	rows = append(rows, []string{}, []string{"degradation_flag"})
	flags := append([]string(nil), pack.DegradationFlags...)
	sort.Strings(flags)
	if len(flags) == 0 {
		rows = append(rows, []string{"none"})
	} else {
		for _, flag := range flags {
			rows = append(rows, []string{flag})
		}
	}

	rows = append(rows, []string{}, []string{"record_id", "record_type", "timestamp", "auditability", "exception_classes"})
	for _, record := range pack.Records {
		rows = append(rows, []string{
			record.RecordID,
			record.RecordType,
			record.Timestamp,
			record.Auditability,
			strings.Join(record.ExceptionClasses, ";"),
		})
	}

	if err := writer.WriteAll(rows); err != nil {
		return nil, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func PDF(pack review.Pack) []byte {
	lines := []string{
		"%PDF-1.4",
		"% Axym Daily Review Pack",
		fmt.Sprintf("date=%s", pack.Date),
		fmt.Sprintf("record_count=%d", pack.RecordCount),
		fmt.Sprintf("empty=%s", boolToString(pack.Empty)),
	}
	for _, item := range pack.Exceptions {
		lines = append(lines, fmt.Sprintf("exception.%s=%d", item.Name, item.Count))
	}
	for _, item := range pack.ReplayTierDistribution {
		lines = append(lines, fmt.Sprintf("replay.%s=%d", item.Name, item.Count))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func boolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
