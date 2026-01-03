package events

// GuardDuty Malware Protection scan result event types.
// These events are published to EventBridge when GuardDuty completes a malware scan.

// GuardDutyEventDetail is the detail portion of an EventBridge event for GuardDuty scan results.
type GuardDutyEventDetail struct {
	SchemaVersion     string            `json:"schemaVersion"`
	ScanStatus        string            `json:"scanStatus"` // "COMPLETED", "SKIPPED"
	ResourceType      string            `json:"resourceType"`
	S3ObjectDetails   S3ObjectDetails   `json:"s3ObjectDetails"`
	ScanResultDetails ScanResultDetails `json:"scanResultDetails,omitempty"`
}

// S3ObjectDetails contains information about the scanned S3 object.
type S3ObjectDetails struct {
	BucketName string `json:"bucketName"`
	ObjectKey  string `json:"objectKey"`
	ETag       string `json:"eTag"`
	VersionID  string `json:"versionId,omitempty"`
}

// ScanResultDetails contains the result of the malware scan.
type ScanResultDetails struct {
	ScanResult string       `json:"scanResult"` // "NO_THREATS_FOUND", "THREATS_FOUND", "UNSUPPORTED", "ACCESS_DENIED", "FAILED"
	Threats    []ThreatInfo `json:"threats,omitempty"`
}

// ThreatInfo contains information about detected threats.
type ThreatInfo struct {
	Name string `json:"name"`
}

// GuardDuty scan result values
const (
	GuardDutyScanResultClean        = "NO_THREATS_FOUND"
	GuardDutyScanResultThreat       = "THREATS_FOUND"
	GuardDutyScanResultUnsupported  = "UNSUPPORTED"
	GuardDutyScanResultAccessDenied = "ACCESS_DENIED"
	GuardDutyScanResultFailed       = "FAILED"
)

// GuardDuty EventBridge source and detail-type
const (
	GuardDutyEventSource     = "aws.guardduty"
	GuardDutyEventDetailType = "GuardDuty Malware Protection Object Scan Result"
)
