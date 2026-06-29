package domain

import "time"

const (
	ImageWatermarkPayloadVersion    = 3
	ImageWatermarkTraceTokenBytes   = 8
	ImageWatermarkPayloadBytes      = 4
	ImageWatermarkMinShortCodeBytes = 2
	ImageWatermarkShortCodeBytes    = 4

	ImageWatermarkTraceUpload    = "upload"
	ImageWatermarkTraceProtected = "protected"

	ImageWatermarkFieldUID        = 1 << 0
	ImageWatermarkFieldUserID     = 1 << 1
	ImageWatermarkFieldUsername   = 1 << 2
	ImageWatermarkFieldTime       = 1 << 3
	ImageWatermarkFieldSourceHash = 1 << 4
	ImageWatermarkFieldCustom     = 1 << 5
)

type ImageWatermarkTrace struct {
	ID              int64      `gorm:"column:id;primaryKey"`
	Token           string     `gorm:"column:token;size:16;uniqueIndex:idx_image_watermark_traces_token"`
	ShortCode       *string    `gorm:"column:short_code;size:8;uniqueIndex:idx_image_watermark_traces_short_code"`
	ShortCodeBytes  int        `gorm:"column:short_code_bytes"`
	TraceType       string     `gorm:"column:trace_type;size:32;index:idx_image_watermark_traces_type_created,priority:1"`
	PayloadVersion  int        `gorm:"column:payload_version"`
	PayloadBytes    int        `gorm:"column:payload_bytes"`
	FieldFlags      int        `gorm:"column:field_flags"`
	Engine          string     `gorm:"column:engine;size:16"`
	WatermarkWidth  int        `gorm:"column:watermark_width"`
	WatermarkHeight int        `gorm:"column:watermark_height"`
	UserID          int64      `gorm:"column:user_id;index:idx_image_watermark_traces_user_created,priority:1"`
	UserDisplayID   string     `gorm:"column:user_display_id;size:128"`
	Username        string     `gorm:"column:username;size:255"`
	PostID          *int64     `gorm:"column:post_id;index:idx_image_watermark_traces_post"`
	ImageID         *int64     `gorm:"column:image_id;index:idx_image_watermark_traces_image;uniqueIndex:idx_image_watermark_traces_job_image,priority:2"`
	JobID           string     `gorm:"column:job_id;size:64;uniqueIndex:idx_image_watermark_traces_job_image,priority:1"`
	SourceURL       string     `gorm:"column:source_url;type:text"`
	SourceHash      string     `gorm:"column:source_hash;size:64"`
	CustomText      string     `gorm:"column:custom_text;size:255"`
	CreatedAt       time.Time  `gorm:"column:created_at;index:idx_image_watermark_traces_type_created,priority:2;index:idx_image_watermark_traces_user_created,priority:2"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

func (ImageWatermarkTrace) TableName() string { return "image_watermark_traces" }
