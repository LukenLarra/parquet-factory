// Copyright 2022 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rulereportaggregator

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/parquet-factory/metrics"
	"github.com/RedHatInsights/parquet-factory/reportaggregators"
	"github.com/RedHatInsights/parquet-factory/s3writer"
	"github.com/RedHatInsights/parquet-factory/utils"
)

const ruleHitsTableName = "rule_hits"

// RuleHitTable is Go representation of single row of rule_hits table
type RuleHitTable struct {
	ClusterID   string `parquet:"name=cluster_id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	RuleID      string `parquet:"name=rule_id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	CollectedAt int64  `parquet:"name=collected_at, type=INT64, logicaltype=TIMESTAMP, logicaltype.isadjustedtoutc=true, logicaltype.unit=MILLIS"`
	ArchivePath string `parquet:"name=archive_path, type=BYTE_ARRAY, encoding=PLAIN"`
}

func (aggregator *RulesResultsReportAggregator) createRuleHitTable(writer s3writer.S3ParquetWriter) ([]string, error) {
	log.Info().Msgf(reportaggregators.StartGenerateFileStr, ruleHitsTableName)

	ctx := context.Background()
	savedFiles := []string{}

	table, err := aggregator.generateRuleHitRows()
	if err != nil {
		log.Error().Err(err).Msgf(reportaggregators.UnableGenerateTableStr, ruleHitsTableName)
		return savedFiles, err
	}

	for timestamp, rows := range table {
		// generate filepath without index first
		ruleHitsPrefix := utils.GenerateHourPrefix(timestamp, writer.Prefix(), ruleHitsTableName)
		indexes := writer.GetLastIndexForParquet(ctx, ruleHitsPrefix)
		fileID, ok := indexes[ruleHitsTableName]
		if !ok {
			fileID = 0
		} else {
			fileID++
		}

		parquetFilePath := aggregator.generateRuleHitsFilePath(timestamp, writer.Prefix(), fileID)
		log.Info().Msgf(reportaggregators.FileStoredStr, parquetFilePath)

		// Init writers directly to bucket
		file, err := writer.NewFile(ctx, parquetFilePath, new(RuleHitTable))
		if err != nil {
			log.Error().Err(err).Msg(reportaggregators.UnableCreateFileStr)
			return savedFiles, err
		}

		for _, row := range rows {
			if err = file.AddRow(row); err != nil {
				log.Error().Err(err).Msgf(reportaggregators.UnableSaveRowStr, ruleHitsTableName)
				continue
			}
			reportaggregators.LogInsertedRow(row.ArchivePath, ruleHitsTableName)
		}

		if err := file.CloseFile(); err != nil {
			log.Error().Err(err).Msg(reportaggregators.UnableCloseFileStr)
			if closingErr := writer.DeleteFiles(savedFiles); closingErr != nil {
				log.Error().Err(closingErr).Msg(reportaggregators.UnableDeleteFileStr)
			}
			return savedFiles, err
		}
		log.Info().Msgf(reportaggregators.GenerateFileSuccess, ruleHitsTableName, fileID)
		savedFiles = append(savedFiles, parquetFilePath)
		metrics.FilesGenerated.With(metrics.WithTableLabel(ruleHitsTableName)).Inc()
	}

	return savedFiles, nil
}

// generateRuleHitsFilePath generates full path of rule_hits parquet file
func (aggregator *RulesResultsReportAggregator) generateRuleHitsFilePath(timestamp time.Time, prefix string, index int) string {
	return utils.GenerateParquetFilepath(timestamp, prefix, ruleHitsTableName, index)
}

func (aggregator *RulesResultsReportAggregator) generateRuleHitRows() (map[time.Time][]RuleHitTable, error) {
	tableRows := map[time.Time][]RuleHitTable{}

	aggregator.mutex.RLock()
	defer aggregator.mutex.RUnlock()

	for _, report := range aggregator.ReceivedReports {
		collectedAt, err := reportaggregators.ExtractCollectedDate(report.Path)
		if err != nil {
			log.Error().
				Err(err).
				Str("archive_path", report.Path).
				Str("cluster_id", report.Metadata.ClusterID).
				Int("info_count", len(report.Report.Info)).
				Int("reports_count", len(report.Report.Reports)).
				Interface("full_report", report).
				Msgf("Unable to find collected at date for report")
			continue
		}
		collectedHour := utils.GetHourOnly(collectedAt)

		// Push new data to parquet table
		for _, ruleReport := range report.Report.Reports {
			tableRows[collectedHour] = append(tableRows[collectedHour], RuleHitTable{
				ClusterID:   report.Metadata.ClusterID,
				RuleID:      ruleReport.RuleID,
				CollectedAt: collectedAt.Unix() * 1000,
				ArchivePath: report.Path,
			})
		}
	}
	return tableRows, nil
}
