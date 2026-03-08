package skills

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
)

// DataAnalyzeResult represents the result of data analysis
type DataAnalyzeResult struct {
	Summary    DataSummary             `json:"summary"`
	Statistics map[string]interface{} `json:"statistics"`
	Patterns   []DataPattern           `json:"patterns"`
	Insights   []string               `json:"insights"`
	Charts     []ChartDefinition      `json:"charts,omitempty"`
	Quality    DataQuality             `json:"quality"`
}

// DataSummary provides a high-level summary of the data
type DataSummary struct {
	TotalRecords   int                    `json:"total_records"`
	TotalFields    int                    `json:"total_fields"`
	DataTypes      map[string]int         `json:"data_types"`
	MemoryUsage   string                 `json:"memory_usage"`
	AnalysisTime  float64                `json:"analysis_time_ms"`
}

// DataPattern represents discovered patterns in the data
type DataPattern struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Fields      []string               `json:"fields"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ChartDefinition represents a chart definition for visualization
type ChartDefinition struct {
	Type   string        `json:"type"`   // "bar", "line", "pie", "scatter"
	Title  string        `json:"title"`
	Data   ChartData     `json:"data"`
	Config ChartConfig   `json:"config,omitempty"`
}

// ChartData contains the data for a chart
type ChartData struct {
	Labels []string      `json:"labels,omitempty"`
	Series []ChartSeries `json:"series"`
	Values []float64     `json:"values,omitempty"`
}

// ChartSeries represents a data series in a chart
type ChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}

// ChartConfig contains chart configuration
type ChartConfig struct {
	XAxis string `json:"x_axis,omitempty"`
	YAxis string `json:"y_axis,omitempty"`
	Theme string `json:"theme,omitempty"`
}

// DataQuality represents data quality metrics
type DataQuality struct {
	Completeness    float64            `json:"completeness"`     // 0.0 to 1.0
	Consistency     float64            `json:"consistency"`      // 0.0 to 1.0
	Accuracy        float64            `json:"accuracy"`         // 0.0 to 1.0
	Uniqueness      float64            `json:"uniqueness"`       // 0.0 to 1.0
	Issues          []DataQualityIssue `json:"issues"`
	Score           float64            `json:"overall_score"`    // 0.0 to 1.0
}

// DataQualityIssue represents a specific data quality issue
type DataQualityIssue struct {
	Type        string `json:"type"`
	Field       string `json:"field"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // "low", "medium", "high"
	Count       int    `json:"count"`
}

// DataAnalyzeParams represents parameters for data analysis
type DataAnalyzeParams struct {
	Data          interface{} `json:"data"`
	AnalysisType  string      `json:"analysis_type,omitempty"`  // "basic", "statistical", "patterns", "full"
	Fields        []string    `json:"fields,omitempty"`        // Specific fields to analyze
	GenerateCharts bool       `json:"generate_charts,omitempty"`
	DetectOutliers bool       `json:"detect_outliers,omitempty"`
}

// ExecuteDataAnalyze performs data analysis
func ExecuteDataAnalyze(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	analysisParams, err := parseDataAnalyzeParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid data analysis parameters: %w", err)
	}

	// Validate parameters
	if err := ValidateDataAnalyzeParams(analysisParams); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Convert data to analyzable format
	dataArray, err := convertToDataArray(analysisParams.Data)
	if err != nil {
		return nil, fmt.Errorf("data conversion failed: %w", err)
	}

	// Perform analysis based on type
	switch analysisParams.AnalysisType {
	case "basic":
		return performBasicAnalysis(ctx, dataArray, analysisParams)
	case "statistical":
		return performStatisticalAnalysis(ctx, dataArray, analysisParams)
	case "patterns":
		return performPatternAnalysis(ctx, dataArray, analysisParams)
	default:
		return performFullAnalysis(ctx, dataArray, analysisParams)
	}
}

// parseDataAnalyzeParams parses data analysis parameters
func parseDataAnalyzeParams(params map[string]interface{}) (*DataAnalyzeParams, error) {
	analysisParams := &DataAnalyzeParams{}

	// Extract required parameters
	if data, ok := params["data"]; ok {
		analysisParams.Data = data
	} else {
		return nil, fmt.Errorf("data parameter is required")
	}

	// Extract optional parameters
	if analysisType, ok := params["analysis_type"].(string); ok {
		analysisParams.AnalysisType = strings.ToLower(analysisType)
	} else {
		analysisParams.AnalysisType = "basic"
	}

	if fields, ok := params["fields"].([]interface{}); ok {
		analysisParams.Fields = make([]string, len(fields))
		for i, field := range fields {
			if fieldStr, ok := field.(string); ok {
				analysisParams.Fields[i] = fieldStr
			}
		}
	}

	if generateCharts, ok := params["generate_charts"].(bool); ok {
		analysisParams.GenerateCharts = generateCharts
	}

	if detectOutliers, ok := params["detect_outliers"].(bool); ok {
		analysisParams.DetectOutliers = detectOutliers
	}

	return analysisParams, nil
}

// convertToDataArray converts input data to a consistent array format
func convertToDataArray(data interface{}) ([]map[string]interface{}, error) {
	switch v := data.(type) {
	case []map[string]interface{}:
		return v, nil
	case []interface{}:
		result := make([]map[string]interface{}, len(v))
		for i, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result[i] = itemMap
			} else {
				return nil, fmt.Errorf("item at index %d is not an object", i)
			}
		}
		return result, nil
	case map[string]interface{}:
		return []map[string]interface{}{v}, nil
	default:
		return nil, fmt.Errorf("unsupported data type: expected array or object")
	}
}

// performBasicAnalysis performs basic data analysis
func performBasicAnalysis(ctx context.Context, data []map[string]interface{}, params *DataAnalyzeParams) (*DataAnalyzeResult, error) {
	result := &DataAnalyzeResult{}

	// Calculate basic summary
	summary := calculateDataSummary(data)
	result.Summary = summary

	// Calculate basic statistics
	statistics := calculateBasicStatistics(data)
	result.Statistics = statistics

	// Assess data quality
	quality := assessDataQuality(data)
	result.Quality = quality

	// Generate basic insights
	insights := generateBasicInsights(data, summary, quality)
	result.Insights = insights

	// Generate patterns (basic)
	patterns := detectBasicPatterns(data)
	result.Patterns = patterns

	// Generate charts if requested
	if params.GenerateCharts {
		charts := generateBasicCharts(data, statistics)
		result.Charts = charts
	}

	return result, nil
}

// performStatisticalAnalysis performs statistical analysis
func performStatisticalAnalysis(ctx context.Context, data []map[string]interface{}, params *DataAnalyzeParams) (*DataAnalyzeResult, error) {
	// Start with basic analysis
	result, err := performBasicAnalysis(ctx, data, params)
	if err != nil {
		return nil, err
	}

	// Add advanced statistics (for Phase 2, this extends basic statistics)
	advancedStats := make(map[string]interface{})
	
	// Add correlation analysis for numeric fields
	numericFields := getNumericFields(data)
	if len(numericFields) >= 2 {
		advancedStats["correlation_analysis"] = "Multiple numeric fields detected - correlation analysis available"
		advancedStats["numeric_field_count"] = len(numericFields)
	}
	
	// Add data distribution info
	if len(data) > 0 {
		advancedStats["data_distribution"] = map[string]interface{}{
			"record_count": len(data),
			"field_diversity": len(getAllFields(data)),
		}
	}
	
	for k, v := range advancedStats {
		result.Statistics[k] = v
	}

	// Detect outliers if requested
	if params.DetectOutliers {
		outliers := detectOutliers(data)
		if len(outliers) > 0 {
			result.Statistics["outliers_detected"] = len(outliers)
			result.Statistics["outlier_details"] = outliers
		}
	}

	return result, nil
}

// performPatternAnalysis performs pattern detection
func performPatternAnalysis(ctx context.Context, data []map[string]interface{}, params *DataAnalyzeParams) (*DataAnalyzeResult, error) {
	// Start with basic analysis
	result, err := performBasicAnalysis(ctx, data, params)
	if err != nil {
		return nil, err
	}

	// Add advanced pattern detection
	advancedPatterns := detectAdvancedPatterns(data)
	result.Patterns = append(result.Patterns, advancedPatterns...)

	return result, nil
}

// performFullAnalysis performs comprehensive analysis
func performFullAnalysis(ctx context.Context, data []map[string]interface{}, params *DataAnalyzeParams) (*DataAnalyzeResult, error) {
	// Perform statistical analysis
	result, err := performStatisticalAnalysis(ctx, data, params)
	if err != nil {
		return nil, err
	}

	// Add pattern analysis
	advancedPatterns := detectAdvancedPatterns(data)
	result.Patterns = append(result.Patterns, advancedPatterns...)

	// Add advanced insights
	advancedInsights := generateAdvancedInsights(data, result.Summary, result.Quality)
	result.Insights = append(result.Insights, advancedInsights...)

	return result, nil
}

// calculateDataSummary calculates basic data summary
func calculateDataSummary(data []map[string]interface{}) DataSummary {
	summary := DataSummary{
		TotalRecords: len(data),
		DataTypes:    make(map[string]int),
	}

	if len(data) == 0 {
		return summary
	}

	// Get all unique fields
	fields := make(map[string]struct{})
	for _, record := range data {
		for field := range record {
			fields[field] = struct{}{}
		}
	}
	summary.TotalFields = len(fields)

	// Count data types
	for _, record := range data {
		for _, value := range record {
			dataType := getDataType(value)
			summary.DataTypes[dataType]++
		}
	}

	// Estimate memory usage (rough approximation)
	totalValues := len(data) * len(fields)
	summary.MemoryUsage = fmt.Sprintf("~%.1fKB", float64(totalValues)*0.1) // Rough estimate

	return summary
}

// calculateBasicStatistics calculates basic statistics
func calculateBasicStatistics(data []map[string]interface{}) map[string]interface{} {
	stats := make(map[string]interface{})
	
	if len(data) == 0 {
		return stats
	}

	// Get all numeric fields
	numericFields := getNumericFields(data)
	
	for _, field := range numericFields {
		values := getNumericValues(data, field)
		if len(values) == 0 {
			continue
		}

		fieldStats := calculateFieldStatistics(values)
		stats[field] = fieldStats
	}

	// Add overall statistics
	stats["total_records"] = len(data)
	stats["numeric_fields"] = len(numericFields)
	stats["text_fields"] = len(getTextFields(data))
	stats["null_values"] = countNullValues(data)

	return stats
}

// calculateFieldStatistics calculates statistics for a numeric field
func calculateFieldStatistics(values []float64) map[string]interface{} {
	if len(values) == 0 {
		return map[string]interface{}{}
	}

	stats := map[string]interface{}{
		"count":  len(values),
		"min":    values[0],
		"max":    values[0],
		"sum":    0.0,
		"mean":   0.0,
		"median": 0.0,
	}

	// Sort values for median calculation
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	// Calculate min, max, sum
	min, max, sum := sortedValues[0], sortedValues[len(sortedValues)-1], 0.0
	for _, v := range sortedValues {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	mean := sum / float64(len(sortedValues))

	// Calculate median
	var median float64
	mid := len(sortedValues) / 2
	if len(sortedValues)%2 == 0 {
		median = (sortedValues[mid-1] + sortedValues[mid]) / 2
	} else {
		median = sortedValues[mid]
	}

	stats["min"] = min
	stats["max"] = max
	stats["sum"] = sum
	stats["mean"] = mean
	stats["median"] = median

	// Calculate standard deviation if we have enough values
	if len(sortedValues) > 1 {
		var variance float64
		for _, v := range sortedValues {
			variance += math.Pow(v-mean, 2)
		}
		variance /= float64(len(sortedValues))
		stdDev := math.Sqrt(variance)
		stats["std_dev"] = stdDev
		stats["variance"] = variance
	}

	return stats
}

// assessDataQuality assesses the quality of the data
func assessDataQuality(data []map[string]interface{}) DataQuality {
	quality := DataQuality{}

	if len(data) == 0 {
		quality.Score = 0.0
		return quality
	}

	totalCells := 0
	nullCells := 0
	emptyCells := 0

	// Get all fields
	fields := getAllFields(data)

	// Check each cell
	for _, record := range data {
		for _, field := range fields {
			totalCells++
			value, exists := record[field]
			if !exists || value == nil {
				nullCells++
				quality.Issues = append(quality.Issues, DataQualityIssue{
					Type:        "missing_value",
					Field:       field,
					Description: "Missing or null value",
					Severity:    "low",
					Count:       1,
				})
			} else if strValue, ok := value.(string); ok && strings.TrimSpace(strValue) == "" {
				emptyCells++
				quality.Issues = append(quality.Issues, DataQualityIssue{
					Type:        "empty_string",
					Field:       field,
					Description: "Empty string value",
					Severity:    "low",
					Count:       1,
				})
			}
		}
	}

	// Calculate completeness (percentage of non-null values)
	if totalCells > 0 {
		quality.Completeness = float64(totalCells-nullCells) / float64(totalCells)
	}

	// Calculate consistency (simplified - checks for data type consistency)
	quality.Consistency = calculateConsistencyScore(data)

	// Calculate uniqueness (simplified - checks for duplicate records)
	quality.Uniqueness = calculateUniquenessScore(data)

	// Accuracy is hard to determine without validation rules, so we use a heuristic
	quality.Accuracy = (quality.Completeness + quality.Consistency) / 2

	// Calculate overall score
	quality.Score = (quality.Completeness + quality.Consistency + quality.Accuracy + quality.Uniqueness) / 4

	return quality
}

// generateBasicInsights generates basic insights from the data
func generateBasicInsights(data []map[string]interface{}, summary DataSummary, quality DataQuality) []string {
	insights := []string{}

	// Size insights
	if summary.TotalRecords == 0 {
		insights = append(insights, "Dataset is empty")
	} else if summary.TotalRecords < 10 {
		insights = append(insights, "Dataset is very small (< 10 records)")
	} else if summary.TotalRecords > 10000 {
		insights = append(insights, "Dataset is very large (> 10,000 records)")
	} else {
		insights = append(insights, fmt.Sprintf("Dataset contains %d records", summary.TotalRecords))
	}

	// Quality insights
	if quality.Score >= 0.9 {
		insights = append(insights, "Data quality is excellent")
	} else if quality.Score >= 0.7 {
		insights = append(insights, "Data quality is good")
	} else if quality.Score >= 0.5 {
		insights = append(insights, "Data quality is fair")
	} else {
		insights = append(insights, "Data quality needs improvement")
	}

	// Data type insights
	if len(summary.DataTypes) > 0 {
		dominantType := ""
		maxCount := 0
		for dataType, count := range summary.DataTypes {
			if count > maxCount {
				maxCount = count
				dominantType = dataType
			}
		}
		insights = append(insights, fmt.Sprintf("Dataset is primarily %s data", dominantType))
	}

	return insights
}

// detectBasicPatterns detects basic patterns in the data
func detectBasicPatterns(data []map[string]interface{}) []DataPattern {
	patterns := []DataPattern{}

	if len(data) == 0 {
		return patterns
	}

	// Detect data type patterns
	types := make(map[string]int)
	for _, record := range data {
		for _, value := range record {
			dataType := getDataType(value)
			types[dataType]++
		}
	}

	// Add data type distribution pattern
	patterns = append(patterns, DataPattern{
		Type:        "data_type_distribution",
		Description: "Dataset contains mixed data types",
		Confidence:  1.0,
		Fields:      []string{},
		Details:     map[string]interface{}{"distribution": types},
	})

	// Detect field presence patterns
	fields := getAllFields(data)
	if len(fields) > 0 {
		patterns = append(patterns, DataPattern{
			Type:        "field_structure",
			Description: fmt.Sprintf("Dataset has %d consistent fields", len(fields)),
			Confidence:  1.0,
			Fields:      fields,
		})
	}

	return patterns
}

// generateBasicCharts generates basic chart definitions
func generateBasicCharts(data []map[string]interface{}, stats map[string]interface{}) []ChartDefinition {
	charts := []ChartDefinition{}

	if len(data) == 0 {
		return charts
	}

	// Generate data type distribution chart
	dataTypeChart := ChartDefinition{
		Type:  "pie",
		Title: "Data Type Distribution",
		Data: ChartData{
			Labels: []string{},
			Values: []float64{},
		},
	}

	types := make(map[string]int)
	for _, record := range data {
		for _, value := range record {
			dataType := getDataType(value)
			types[dataType]++
		}
	}

	for dataType, count := range types {
		dataTypeChart.Data.Labels = append(dataTypeChart.Data.Labels, dataType)
		dataTypeChart.Data.Values = append(dataTypeChart.Data.Values, float64(count))
	}

	if len(dataTypeChart.Data.Labels) > 0 {
		charts = append(charts, dataTypeChart)
	}

	// Generate numeric field statistics chart if available
	numericFields := getNumericFields(data)
	if len(numericFields) > 0 {
		fieldStats := make([]ChartSeries, 0, len(numericFields))
		
		for _, field := range numericFields {
			if fieldStatsData, ok := stats[field].(map[string]interface{}); ok {
				if mean, ok := fieldStatsData["mean"].(float64); ok {
					fieldStats = append(fieldStats, ChartSeries{
						Name:   field,
						Values: []float64{mean},
						Color:  "#4CAF50",
					})
				}
			}
		}

		if len(fieldStats) > 0 {
			charts = append(charts, ChartDefinition{
				Type:  "bar",
				Title: "Numeric Field Means",
				Data: ChartData{
					Labels: []string{"Mean Values"},
					Series: fieldStats,
				},
			})
		}
	}

	return charts
}

// Helper functions

func getDataType(value interface{}) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

func getNumericFields(data []map[string]interface{}) []string {
	if len(data) == 0 {
		return []string{}
	}

	numericFields := make(map[string]struct{})
	for _, record := range data {
		for field, value := range record {
			switch value.(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
				numericFields[field] = struct{}{}
			}
		}
	}

	fields := make([]string, 0, len(numericFields))
	for field := range numericFields {
		fields = append(fields, field)
	}
	return fields
}

func getTextFields(data []map[string]interface{}) []string {
	if len(data) == 0 {
		return []string{}
	}

	textFields := make(map[string]struct{})
	for _, record := range data {
		for field, value := range record {
			if _, ok := value.(string); ok {
				textFields[field] = struct{}{}
			}
		}
	}

	fields := make([]string, 0, len(textFields))
	for field := range textFields {
		fields = append(fields, field)
	}
	return fields
}

func getAllFields(data []map[string]interface{}) []string {
	if len(data) == 0 {
		return []string{}
	}

	fields := make(map[string]struct{})
	for _, record := range data {
		for field := range record {
			fields[field] = struct{}{}
		}
	}

	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	return result
}

func getNumericValues(data []map[string]interface{}, field string) []float64 {
	values := []float64{}
	for _, record := range data {
		if value, exists := record[field]; exists {
			switch v := value.(type) {
			case int:
				values = append(values, float64(v))
			case int8:
				values = append(values, float64(v))
			case int16:
				values = append(values, float64(v))
			case int32:
				values = append(values, float64(v))
			case int64:
				values = append(values, float64(v))
			case uint:
				values = append(values, float64(v))
			case uint8:
				values = append(values, float64(v))
			case uint16:
				values = append(values, float64(v))
			case uint32:
				values = append(values, float64(v))
			case uint64:
				values = append(values, float64(v))
			case float32:
				values = append(values, float64(v))
			case float64:
				values = append(values, v)
			}
		}
	}
	return values
}

func countNullValues(data []map[string]interface{}) int {
	count := 0
	for _, record := range data {
		for _, value := range record {
			if value == nil {
				count++
			}
		}
	}
	return count
}

func calculateConsistencyScore(data []map[string]interface{}) float64 {
	if len(data) == 0 {
		return 0.0
	}

	fields := getAllFields(data)
	if len(fields) == 0 {
		return 1.0
	}

	consistentFields := 0
	for _, field := range fields {
		types := make(map[string]int)
		for _, record := range data {
			if value, exists := record[field]; exists {
				dataType := getDataType(value)
				types[dataType]++
			}
		}
		// Field is consistent if all values are of the same type or null
		if len(types) <= 1 {
			consistentFields++
		}
	}

	return float64(consistentFields) / float64(len(fields))
}

func calculateUniquenessScore(data []map[string]interface{}) float64 {
	if len(data) <= 1 {
		return 1.0
	}

	// Simple duplicate detection based on string representation
	seen := make(map[string]bool)
	unique := 0
	for _, record := range data {
		recordStr := fmt.Sprintf("%v", record)
		if !seen[recordStr] {
			seen[recordStr] = true
			unique++
		}
	}

	return float64(unique) / float64(len(data))
}

func detectAdvancedPatterns(data []map[string]interface{}) []DataPattern {
	// This is a simplified implementation for Phase 2
	// In production, you would add more sophisticated pattern detection algorithms
	patterns := []DataPattern{}

	// Detect numeric correlations (simplified)
	numericFields := getNumericFields(data)
	if len(numericFields) >= 2 {
		patterns = append(patterns, DataPattern{
			Type:        "numeric_correlation",
			Description: "Multiple numeric fields detected - may have correlations",
			Confidence:  0.7,
			Fields:      numericFields,
		})
	}

	return patterns
}

func generateAdvancedInsights(data []map[string]interface{}, summary DataSummary, quality DataQuality) []string {
	insights := []string{}

	// Advanced insights based on data characteristics
	if summary.TotalRecords > 1000 {
		insights = append(insights, "Large dataset - consider sampling for analysis")
	}

	if quality.Completeness < 0.8 {
		insights = append(insights, "Data completeness is low - consider data cleaning")
	}

	numericFields := getNumericFields(data)
	if len(numericFields) > len(getTextFields(data)) {
		insights = append(insights, "Dataset is numeric-heavy - suitable for statistical analysis")
	} else {
		insights = append(insights, "Dataset is text-heavy - suitable for text analysis")
	}

	return insights
}

func detectOutliers(data []map[string]interface{}) []map[string]interface{} {
	// Simplified outlier detection using IQR method
	outliers := []map[string]interface{}{}

	numericFields := getNumericFields(data)
	for _, field := range numericFields {
		values := getNumericValues(data, field)
		if len(values) < 4 {
			continue // Need at least 4 values for IQR
		}

		// Sort values
		sorted := make([]float64, len(values))
		copy(sorted, values)
		sort.Float64s(sorted)

		// Calculate Q1, Q3, and IQR
		q1Index := len(sorted) / 4
		q3Index := len(sorted) * 3 / 4
		q1 := sorted[q1Index]
		q3 := sorted[q3Index]
		iqr := q3 - q1

		// Define outlier bounds
		lowerBound := q1 - 1.5*iqr
		upperBound := q3 + 1.5*iqr

		// Find outliers
		for i, record := range data {
			if value, exists := record[field]; exists {
				if numValue, ok := toFloat64(value); ok {
					if numValue < lowerBound || numValue > upperBound {
						outliers = append(outliers, map[string]interface{}{
							"record_index": i,
							"field":        field,
							"value":        numValue,
							"lower_bound":  lowerBound,
							"upper_bound":  upperBound,
						})
					}
				}
			}
		}
	}

	return outliers
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

// ValidateDataAnalyzeParams validates data analysis parameters
func ValidateDataAnalyzeParams(params *DataAnalyzeParams) error {
	if params.Data == nil {
		return fmt.Errorf("data parameter is required")
	}

	// Validate analysis type
	if params.AnalysisType != "" {
		allowedTypes := []string{"basic", "statistical", "patterns", "full"}
		allowed := false
		for _, allowedType := range allowedTypes {
			if params.AnalysisType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("unsupported analysis type: %s (supported: basic, statistical, patterns, full)", params.AnalysisType)
		}
	}

	return nil
}