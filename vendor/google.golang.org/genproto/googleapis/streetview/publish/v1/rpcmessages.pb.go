// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/streetview/publish/v1/rpcmessages.proto

package publish

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	status "google.golang.org/genproto/googleapis/rpc/status"
	field_mask "google.golang.org/genproto/protobuf/field_mask"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Specifies which view of the `Photo` should be included in the response.
type PhotoView int32

const (
	// Server reponses do not include the download URL for the photo bytes.
	// The default value.
	PhotoView_BASIC PhotoView = 0
	// Server responses include the download URL for the photo bytes.
	PhotoView_INCLUDE_DOWNLOAD_URL PhotoView = 1
)

var PhotoView_name = map[int32]string{
	0: "BASIC",
	1: "INCLUDE_DOWNLOAD_URL",
}

var PhotoView_value = map[string]int32{
	"BASIC":                0,
	"INCLUDE_DOWNLOAD_URL": 1,
}

func (x PhotoView) String() string {
	return proto.EnumName(PhotoView_name, int32(x))
}

func (PhotoView) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{0}
}

// Request to create a photo.
type CreatePhotoRequest struct {
	// Required. Photo to create.
	Photo                *Photo   `protobuf:"bytes,1,opt,name=photo,proto3" json:"photo,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreatePhotoRequest) Reset()         { *m = CreatePhotoRequest{} }
func (m *CreatePhotoRequest) String() string { return proto.CompactTextString(m) }
func (*CreatePhotoRequest) ProtoMessage()    {}
func (*CreatePhotoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{0}
}

func (m *CreatePhotoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreatePhotoRequest.Unmarshal(m, b)
}
func (m *CreatePhotoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreatePhotoRequest.Marshal(b, m, deterministic)
}
func (m *CreatePhotoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreatePhotoRequest.Merge(m, src)
}
func (m *CreatePhotoRequest) XXX_Size() int {
	return xxx_messageInfo_CreatePhotoRequest.Size(m)
}
func (m *CreatePhotoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreatePhotoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreatePhotoRequest proto.InternalMessageInfo

func (m *CreatePhotoRequest) GetPhoto() *Photo {
	if m != nil {
		return m.Photo
	}
	return nil
}

// Request to get a photo.
//
// By default
// - does not return the download URL for the photo bytes.
//
// Parameters:
// - 'view' controls if the download URL for the photo bytes will be returned.
type GetPhotoRequest struct {
	// Required. ID of the photo.
	PhotoId string `protobuf:"bytes,1,opt,name=photo_id,json=photoId,proto3" json:"photo_id,omitempty"`
	// Specifies if a download URL for the photo bytes should be returned in the
	// Photo response.
	View                 PhotoView `protobuf:"varint,2,opt,name=view,proto3,enum=google.streetview.publish.v1.PhotoView" json:"view,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *GetPhotoRequest) Reset()         { *m = GetPhotoRequest{} }
func (m *GetPhotoRequest) String() string { return proto.CompactTextString(m) }
func (*GetPhotoRequest) ProtoMessage()    {}
func (*GetPhotoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{1}
}

func (m *GetPhotoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetPhotoRequest.Unmarshal(m, b)
}
func (m *GetPhotoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetPhotoRequest.Marshal(b, m, deterministic)
}
func (m *GetPhotoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetPhotoRequest.Merge(m, src)
}
func (m *GetPhotoRequest) XXX_Size() int {
	return xxx_messageInfo_GetPhotoRequest.Size(m)
}
func (m *GetPhotoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetPhotoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetPhotoRequest proto.InternalMessageInfo

func (m *GetPhotoRequest) GetPhotoId() string {
	if m != nil {
		return m.PhotoId
	}
	return ""
}

func (m *GetPhotoRequest) GetView() PhotoView {
	if m != nil {
		return m.View
	}
	return PhotoView_BASIC
}

// Request to get one or more photos.
// By default
// - does not return the download URL for the photo bytes.
//
// Parameters:
// - 'view' controls if the download URL for the photo bytes will be returned.
type BatchGetPhotosRequest struct {
	// Required. IDs of the photos.
	PhotoIds []string `protobuf:"bytes,1,rep,name=photo_ids,json=photoIds,proto3" json:"photo_ids,omitempty"`
	// Specifies if a download URL for the photo bytes should be returned in the
	// Photo response.
	View                 PhotoView `protobuf:"varint,2,opt,name=view,proto3,enum=google.streetview.publish.v1.PhotoView" json:"view,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *BatchGetPhotosRequest) Reset()         { *m = BatchGetPhotosRequest{} }
func (m *BatchGetPhotosRequest) String() string { return proto.CompactTextString(m) }
func (*BatchGetPhotosRequest) ProtoMessage()    {}
func (*BatchGetPhotosRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{2}
}

func (m *BatchGetPhotosRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchGetPhotosRequest.Unmarshal(m, b)
}
func (m *BatchGetPhotosRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchGetPhotosRequest.Marshal(b, m, deterministic)
}
func (m *BatchGetPhotosRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchGetPhotosRequest.Merge(m, src)
}
func (m *BatchGetPhotosRequest) XXX_Size() int {
	return xxx_messageInfo_BatchGetPhotosRequest.Size(m)
}
func (m *BatchGetPhotosRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchGetPhotosRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BatchGetPhotosRequest proto.InternalMessageInfo

func (m *BatchGetPhotosRequest) GetPhotoIds() []string {
	if m != nil {
		return m.PhotoIds
	}
	return nil
}

func (m *BatchGetPhotosRequest) GetView() PhotoView {
	if m != nil {
		return m.View
	}
	return PhotoView_BASIC
}

// Response to batch get of photos.
type BatchGetPhotosResponse struct {
	// List of results for each individual photo requested, in the same order as
	// the request.
	Results              []*PhotoResponse `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *BatchGetPhotosResponse) Reset()         { *m = BatchGetPhotosResponse{} }
func (m *BatchGetPhotosResponse) String() string { return proto.CompactTextString(m) }
func (*BatchGetPhotosResponse) ProtoMessage()    {}
func (*BatchGetPhotosResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{3}
}

func (m *BatchGetPhotosResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchGetPhotosResponse.Unmarshal(m, b)
}
func (m *BatchGetPhotosResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchGetPhotosResponse.Marshal(b, m, deterministic)
}
func (m *BatchGetPhotosResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchGetPhotosResponse.Merge(m, src)
}
func (m *BatchGetPhotosResponse) XXX_Size() int {
	return xxx_messageInfo_BatchGetPhotosResponse.Size(m)
}
func (m *BatchGetPhotosResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchGetPhotosResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BatchGetPhotosResponse proto.InternalMessageInfo

func (m *BatchGetPhotosResponse) GetResults() []*PhotoResponse {
	if m != nil {
		return m.Results
	}
	return nil
}

// Response payload for a single `Photo` in batch operations including
// `BatchGetPhotosRequest` and `BatchUpdatePhotosRequest`.
type PhotoResponse struct {
	// The status for the operation to get or update a single photo in the batch
	// request.
	Status *status.Status `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	// The photo resource, if the request was successful.
	Photo                *Photo   `protobuf:"bytes,2,opt,name=photo,proto3" json:"photo,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PhotoResponse) Reset()         { *m = PhotoResponse{} }
func (m *PhotoResponse) String() string { return proto.CompactTextString(m) }
func (*PhotoResponse) ProtoMessage()    {}
func (*PhotoResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{4}
}

func (m *PhotoResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PhotoResponse.Unmarshal(m, b)
}
func (m *PhotoResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PhotoResponse.Marshal(b, m, deterministic)
}
func (m *PhotoResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PhotoResponse.Merge(m, src)
}
func (m *PhotoResponse) XXX_Size() int {
	return xxx_messageInfo_PhotoResponse.Size(m)
}
func (m *PhotoResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_PhotoResponse.DiscardUnknown(m)
}

var xxx_messageInfo_PhotoResponse proto.InternalMessageInfo

func (m *PhotoResponse) GetStatus() *status.Status {
	if m != nil {
		return m.Status
	}
	return nil
}

func (m *PhotoResponse) GetPhoto() *Photo {
	if m != nil {
		return m.Photo
	}
	return nil
}

// Request to list all photos that belong to the user sending the request.
//
// By default
// - does not return the download URL for the photo bytes.
//
// Parameters:
// - 'view' controls if the download URL for the photo bytes will be returned.
// - 'page_size' determines the maximum number of photos to return.
// - 'page_token' is the next page token value returned from a previous List
//     request, if any.
type ListPhotosRequest struct {
	// Specifies if a download URL for the photos bytes should be returned in the
	// Photos response.
	View PhotoView `protobuf:"varint,1,opt,name=view,proto3,enum=google.streetview.publish.v1.PhotoView" json:"view,omitempty"`
	// The maximum number of photos to return.
	// `page_size` must be non-negative. If `page_size` is zero or is not
	// provided, the default page size of 100 will be used.
	// The number of photos returned in the response may be less than `page_size`
	// if the number of photos that belong to the user is less than `page_size`.
	PageSize int32 `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	// The next_page_token value returned from a previous List request, if any.
	PageToken string `protobuf:"bytes,3,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
	// The filter expression.
	// Example: `placeId=ChIJj61dQgK6j4AR4GeTYWZsKWw`
	Filter               string   `protobuf:"bytes,4,opt,name=filter,proto3" json:"filter,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListPhotosRequest) Reset()         { *m = ListPhotosRequest{} }
func (m *ListPhotosRequest) String() string { return proto.CompactTextString(m) }
func (*ListPhotosRequest) ProtoMessage()    {}
func (*ListPhotosRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{5}
}

func (m *ListPhotosRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListPhotosRequest.Unmarshal(m, b)
}
func (m *ListPhotosRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListPhotosRequest.Marshal(b, m, deterministic)
}
func (m *ListPhotosRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListPhotosRequest.Merge(m, src)
}
func (m *ListPhotosRequest) XXX_Size() int {
	return xxx_messageInfo_ListPhotosRequest.Size(m)
}
func (m *ListPhotosRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListPhotosRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListPhotosRequest proto.InternalMessageInfo

func (m *ListPhotosRequest) GetView() PhotoView {
	if m != nil {
		return m.View
	}
	return PhotoView_BASIC
}

func (m *ListPhotosRequest) GetPageSize() int32 {
	if m != nil {
		return m.PageSize
	}
	return 0
}

func (m *ListPhotosRequest) GetPageToken() string {
	if m != nil {
		return m.PageToken
	}
	return ""
}

func (m *ListPhotosRequest) GetFilter() string {
	if m != nil {
		return m.Filter
	}
	return ""
}

// Response to list all photos that belong to a user.
type ListPhotosResponse struct {
	// List of photos. There will be a maximum number of items returned based on
	// the page_size field in the request.
	Photos []*Photo `protobuf:"bytes,1,rep,name=photos,proto3" json:"photos,omitempty"`
	// Token to retrieve the next page of results, or empty if there are no
	// more results in the list.
	NextPageToken        string   `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListPhotosResponse) Reset()         { *m = ListPhotosResponse{} }
func (m *ListPhotosResponse) String() string { return proto.CompactTextString(m) }
func (*ListPhotosResponse) ProtoMessage()    {}
func (*ListPhotosResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{6}
}

func (m *ListPhotosResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListPhotosResponse.Unmarshal(m, b)
}
func (m *ListPhotosResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListPhotosResponse.Marshal(b, m, deterministic)
}
func (m *ListPhotosResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListPhotosResponse.Merge(m, src)
}
func (m *ListPhotosResponse) XXX_Size() int {
	return xxx_messageInfo_ListPhotosResponse.Size(m)
}
func (m *ListPhotosResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListPhotosResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListPhotosResponse proto.InternalMessageInfo

func (m *ListPhotosResponse) GetPhotos() []*Photo {
	if m != nil {
		return m.Photos
	}
	return nil
}

func (m *ListPhotosResponse) GetNextPageToken() string {
	if m != nil {
		return m.NextPageToken
	}
	return ""
}

// Request to update the metadata of a photo.
// Updating the pixels of a photo is not supported.
type UpdatePhotoRequest struct {
	// Required. Photo object containing the new metadata. Only the fields
	// specified in `update_mask` are used. If `update_mask` is not present, the
	// update applies to all fields.
	// **Note:** To update `pose.altitude`, `pose.latlngpair` has to be filled as
	// well. Otherwise, the request will fail.
	Photo *Photo `protobuf:"bytes,1,opt,name=photo,proto3" json:"photo,omitempty"`
	// Mask that identifies fields on the photo metadata to update.
	// If not present, the old Photo metadata will be entirely replaced with the
	// new Photo metadata in this request. The update fails if invalid fields are
	// specified. Multiple fields can be specified in a comma-delimited list.
	//
	// The following fields are valid:
	//
	// * `pose.heading`
	// * `pose.latlngpair`
	// * `pose.pitch`
	// * `pose.roll`
	// * `pose.level`
	// * `pose.altitude`
	// * `connections`
	// * `places`
	//
	//
	// **Note:** Repeated fields in `update_mask` mean the entire set of repeated
	// values will be replaced with the new contents. For example, if
	// `UpdatePhotoRequest.photo.update_mask` contains `connections` and
	// `UpdatePhotoRequest.photo.connections` is empty, all connections will be
	// removed.
	UpdateMask           *field_mask.FieldMask `protobuf:"bytes,2,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *UpdatePhotoRequest) Reset()         { *m = UpdatePhotoRequest{} }
func (m *UpdatePhotoRequest) String() string { return proto.CompactTextString(m) }
func (*UpdatePhotoRequest) ProtoMessage()    {}
func (*UpdatePhotoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{7}
}

func (m *UpdatePhotoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UpdatePhotoRequest.Unmarshal(m, b)
}
func (m *UpdatePhotoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UpdatePhotoRequest.Marshal(b, m, deterministic)
}
func (m *UpdatePhotoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UpdatePhotoRequest.Merge(m, src)
}
func (m *UpdatePhotoRequest) XXX_Size() int {
	return xxx_messageInfo_UpdatePhotoRequest.Size(m)
}
func (m *UpdatePhotoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_UpdatePhotoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_UpdatePhotoRequest proto.InternalMessageInfo

func (m *UpdatePhotoRequest) GetPhoto() *Photo {
	if m != nil {
		return m.Photo
	}
	return nil
}

func (m *UpdatePhotoRequest) GetUpdateMask() *field_mask.FieldMask {
	if m != nil {
		return m.UpdateMask
	}
	return nil
}

// Request to update the metadata of photos.
// Updating the pixels of photos is not supported.
type BatchUpdatePhotosRequest struct {
	// Required. List of update photo requests.
	UpdatePhotoRequests  []*UpdatePhotoRequest `protobuf:"bytes,1,rep,name=update_photo_requests,json=updatePhotoRequests,proto3" json:"update_photo_requests,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *BatchUpdatePhotosRequest) Reset()         { *m = BatchUpdatePhotosRequest{} }
func (m *BatchUpdatePhotosRequest) String() string { return proto.CompactTextString(m) }
func (*BatchUpdatePhotosRequest) ProtoMessage()    {}
func (*BatchUpdatePhotosRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{8}
}

func (m *BatchUpdatePhotosRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchUpdatePhotosRequest.Unmarshal(m, b)
}
func (m *BatchUpdatePhotosRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchUpdatePhotosRequest.Marshal(b, m, deterministic)
}
func (m *BatchUpdatePhotosRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchUpdatePhotosRequest.Merge(m, src)
}
func (m *BatchUpdatePhotosRequest) XXX_Size() int {
	return xxx_messageInfo_BatchUpdatePhotosRequest.Size(m)
}
func (m *BatchUpdatePhotosRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchUpdatePhotosRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BatchUpdatePhotosRequest proto.InternalMessageInfo

func (m *BatchUpdatePhotosRequest) GetUpdatePhotoRequests() []*UpdatePhotoRequest {
	if m != nil {
		return m.UpdatePhotoRequests
	}
	return nil
}

// Response to batch update of metadata of one or more photos.
type BatchUpdatePhotosResponse struct {
	// List of results for each individual photo updated, in the same order as
	// the request.
	Results              []*PhotoResponse `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *BatchUpdatePhotosResponse) Reset()         { *m = BatchUpdatePhotosResponse{} }
func (m *BatchUpdatePhotosResponse) String() string { return proto.CompactTextString(m) }
func (*BatchUpdatePhotosResponse) ProtoMessage()    {}
func (*BatchUpdatePhotosResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{9}
}

func (m *BatchUpdatePhotosResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchUpdatePhotosResponse.Unmarshal(m, b)
}
func (m *BatchUpdatePhotosResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchUpdatePhotosResponse.Marshal(b, m, deterministic)
}
func (m *BatchUpdatePhotosResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchUpdatePhotosResponse.Merge(m, src)
}
func (m *BatchUpdatePhotosResponse) XXX_Size() int {
	return xxx_messageInfo_BatchUpdatePhotosResponse.Size(m)
}
func (m *BatchUpdatePhotosResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchUpdatePhotosResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BatchUpdatePhotosResponse proto.InternalMessageInfo

func (m *BatchUpdatePhotosResponse) GetResults() []*PhotoResponse {
	if m != nil {
		return m.Results
	}
	return nil
}

// Request to delete a photo.
type DeletePhotoRequest struct {
	// Required. ID of the photo.
	PhotoId              string   `protobuf:"bytes,1,opt,name=photo_id,json=photoId,proto3" json:"photo_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeletePhotoRequest) Reset()         { *m = DeletePhotoRequest{} }
func (m *DeletePhotoRequest) String() string { return proto.CompactTextString(m) }
func (*DeletePhotoRequest) ProtoMessage()    {}
func (*DeletePhotoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{10}
}

func (m *DeletePhotoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeletePhotoRequest.Unmarshal(m, b)
}
func (m *DeletePhotoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeletePhotoRequest.Marshal(b, m, deterministic)
}
func (m *DeletePhotoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeletePhotoRequest.Merge(m, src)
}
func (m *DeletePhotoRequest) XXX_Size() int {
	return xxx_messageInfo_DeletePhotoRequest.Size(m)
}
func (m *DeletePhotoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeletePhotoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeletePhotoRequest proto.InternalMessageInfo

func (m *DeletePhotoRequest) GetPhotoId() string {
	if m != nil {
		return m.PhotoId
	}
	return ""
}

// Request to delete multiple photos.
type BatchDeletePhotosRequest struct {
	// Required. List of delete photo requests.
	PhotoIds             []string `protobuf:"bytes,1,rep,name=photo_ids,json=photoIds,proto3" json:"photo_ids,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BatchDeletePhotosRequest) Reset()         { *m = BatchDeletePhotosRequest{} }
func (m *BatchDeletePhotosRequest) String() string { return proto.CompactTextString(m) }
func (*BatchDeletePhotosRequest) ProtoMessage()    {}
func (*BatchDeletePhotosRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{11}
}

func (m *BatchDeletePhotosRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchDeletePhotosRequest.Unmarshal(m, b)
}
func (m *BatchDeletePhotosRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchDeletePhotosRequest.Marshal(b, m, deterministic)
}
func (m *BatchDeletePhotosRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchDeletePhotosRequest.Merge(m, src)
}
func (m *BatchDeletePhotosRequest) XXX_Size() int {
	return xxx_messageInfo_BatchDeletePhotosRequest.Size(m)
}
func (m *BatchDeletePhotosRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchDeletePhotosRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BatchDeletePhotosRequest proto.InternalMessageInfo

func (m *BatchDeletePhotosRequest) GetPhotoIds() []string {
	if m != nil {
		return m.PhotoIds
	}
	return nil
}

// Response to batch delete of one or more photos.
type BatchDeletePhotosResponse struct {
	// The status for the operation to delete a single photo in the batch request.
	Status               []*status.Status `protobuf:"bytes,1,rep,name=status,proto3" json:"status,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *BatchDeletePhotosResponse) Reset()         { *m = BatchDeletePhotosResponse{} }
func (m *BatchDeletePhotosResponse) String() string { return proto.CompactTextString(m) }
func (*BatchDeletePhotosResponse) ProtoMessage()    {}
func (*BatchDeletePhotosResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_e56ff94407a6aca7, []int{12}
}

func (m *BatchDeletePhotosResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchDeletePhotosResponse.Unmarshal(m, b)
}
func (m *BatchDeletePhotosResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchDeletePhotosResponse.Marshal(b, m, deterministic)
}
func (m *BatchDeletePhotosResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchDeletePhotosResponse.Merge(m, src)
}
func (m *BatchDeletePhotosResponse) XXX_Size() int {
	return xxx_messageInfo_BatchDeletePhotosResponse.Size(m)
}
func (m *BatchDeletePhotosResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchDeletePhotosResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BatchDeletePhotosResponse proto.InternalMessageInfo

func (m *BatchDeletePhotosResponse) GetStatus() []*status.Status {
	if m != nil {
		return m.Status
	}
	return nil
}

func init() {
	proto.RegisterEnum("google.streetview.publish.v1.PhotoView", PhotoView_name, PhotoView_value)
	proto.RegisterType((*CreatePhotoRequest)(nil), "google.streetview.publish.v1.CreatePhotoRequest")
	proto.RegisterType((*GetPhotoRequest)(nil), "google.streetview.publish.v1.GetPhotoRequest")
	proto.RegisterType((*BatchGetPhotosRequest)(nil), "google.streetview.publish.v1.BatchGetPhotosRequest")
	proto.RegisterType((*BatchGetPhotosResponse)(nil), "google.streetview.publish.v1.BatchGetPhotosResponse")
	proto.RegisterType((*PhotoResponse)(nil), "google.streetview.publish.v1.PhotoResponse")
	proto.RegisterType((*ListPhotosRequest)(nil), "google.streetview.publish.v1.ListPhotosRequest")
	proto.RegisterType((*ListPhotosResponse)(nil), "google.streetview.publish.v1.ListPhotosResponse")
	proto.RegisterType((*UpdatePhotoRequest)(nil), "google.streetview.publish.v1.UpdatePhotoRequest")
	proto.RegisterType((*BatchUpdatePhotosRequest)(nil), "google.streetview.publish.v1.BatchUpdatePhotosRequest")
	proto.RegisterType((*BatchUpdatePhotosResponse)(nil), "google.streetview.publish.v1.BatchUpdatePhotosResponse")
	proto.RegisterType((*DeletePhotoRequest)(nil), "google.streetview.publish.v1.DeletePhotoRequest")
	proto.RegisterType((*BatchDeletePhotosRequest)(nil), "google.streetview.publish.v1.BatchDeletePhotosRequest")
	proto.RegisterType((*BatchDeletePhotosResponse)(nil), "google.streetview.publish.v1.BatchDeletePhotosResponse")
}

func init() {
	proto.RegisterFile("google/streetview/publish/v1/rpcmessages.proto", fileDescriptor_e56ff94407a6aca7)
}

var fileDescriptor_e56ff94407a6aca7 = []byte{
	// 639 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x55, 0xcb, 0x6e, 0xd3, 0x40,
	0x14, 0xc5, 0x7d, 0xa4, 0xcd, 0xad, 0x4a, 0xcb, 0x40, 0x8b, 0x1b, 0x8a, 0x14, 0x19, 0x09, 0xa2,
	0x82, 0xec, 0xb6, 0x2c, 0x10, 0xca, 0xaa, 0x49, 0x4a, 0x55, 0x29, 0x7d, 0xc8, 0xa1, 0x20, 0xb1,
	0xb1, 0x1c, 0xe7, 0xc6, 0xb1, 0xe2, 0x64, 0x5c, 0xcf, 0x38, 0x85, 0xae, 0xf8, 0x00, 0xf8, 0x0b,
	0x3e, 0x14, 0x79, 0x3c, 0xd3, 0x26, 0x69, 0x88, 0x02, 0x74, 0x67, 0xdf, 0xc7, 0xb9, 0x67, 0xce,
	0x9d, 0x63, 0x83, 0xe9, 0x53, 0xea, 0x87, 0x68, 0x31, 0x1e, 0x23, 0xf2, 0x41, 0x80, 0x57, 0x56,
	0x94, 0x34, 0xc3, 0x80, 0x75, 0xac, 0xc1, 0x9e, 0x15, 0x47, 0x5e, 0x0f, 0x19, 0x73, 0x7d, 0x64,
	0x66, 0x14, 0x53, 0x4e, 0xc9, 0x76, 0x56, 0x6f, 0xde, 0xd6, 0x9b, 0xb2, 0xde, 0x1c, 0xec, 0x15,
	0x8a, 0x12, 0x4d, 0xd4, 0x36, 0x93, 0xb6, 0xd5, 0x0e, 0x30, 0x6c, 0x39, 0x3d, 0x97, 0x75, 0xb3,
	0xfe, 0xc2, 0x53, 0x59, 0x11, 0x47, 0x9e, 0xc5, 0xb8, 0xcb, 0x13, 0x09, 0x5c, 0x78, 0x33, 0x9d,
	0x08, 0x32, 0x9a, 0xc4, 0x9e, 0xa2, 0x61, 0x9c, 0x01, 0xa9, 0xc6, 0xe8, 0x72, 0x3c, 0xef, 0x50,
	0x4e, 0x6d, 0xbc, 0x4c, 0x90, 0x71, 0xf2, 0x1e, 0x16, 0xa3, 0xf4, 0x5d, 0xd7, 0x8a, 0x5a, 0x69,
	0x65, 0xff, 0x85, 0x39, 0x8d, 0xac, 0x99, 0xb5, 0x66, 0x1d, 0x46, 0x00, 0x6b, 0x47, 0xc8, 0x47,
	0xd0, 0xb6, 0x60, 0x59, 0xe4, 0x9c, 0xa0, 0x25, 0x00, 0xf3, 0xf6, 0x92, 0x78, 0x3f, 0x6e, 0x91,
	0x32, 0x2c, 0xa4, 0x68, 0xfa, 0x5c, 0x51, 0x2b, 0x3d, 0xdc, 0x7f, 0x35, 0xc3, 0x9c, 0x4f, 0x01,
	0x5e, 0xd9, 0xa2, 0xc9, 0xb8, 0x84, 0x8d, 0x8a, 0xcb, 0xbd, 0x8e, 0x9a, 0xc7, 0xd4, 0xc0, 0x67,
	0x90, 0x57, 0x03, 0x99, 0xae, 0x15, 0xe7, 0x4b, 0x79, 0x7b, 0x59, 0x4e, 0x64, 0xff, 0x37, 0xd2,
	0x81, 0xcd, 0xf1, 0x91, 0x2c, 0xa2, 0x7d, 0x86, 0xe4, 0x10, 0x96, 0x62, 0x64, 0x49, 0xc8, 0xb3,
	0x89, 0x2b, 0xfb, 0xaf, 0x67, 0x11, 0x4d, 0x76, 0xdb, 0xaa, 0xd7, 0x18, 0xc0, 0xea, 0x48, 0x86,
	0xec, 0x40, 0x2e, 0x5b, 0xaf, 0xdc, 0x05, 0x51, 0xb0, 0x71, 0xe4, 0x99, 0x0d, 0x91, 0xb1, 0x65,
	0xc5, 0xed, 0xda, 0xe6, 0xfe, 0x7a, 0x6d, 0xbf, 0x34, 0x78, 0x54, 0x0f, 0xd8, 0x98, 0x90, 0x4a,
	0x2b, 0xed, 0x1f, 0xb4, 0x12, 0x5b, 0x70, 0x7d, 0x74, 0x58, 0x70, 0x8d, 0x82, 0xd1, 0xa2, 0xbd,
	0x9c, 0x06, 0x1a, 0xc1, 0x35, 0x92, 0xe7, 0x00, 0x22, 0xc9, 0x69, 0x17, 0xfb, 0xfa, 0xbc, 0xb8,
	0x15, 0xa2, 0xfc, 0x63, 0x1a, 0x20, 0x9b, 0x90, 0x6b, 0x07, 0x21, 0xc7, 0x58, 0x5f, 0x10, 0x29,
	0xf9, 0x66, 0x7c, 0x03, 0x32, 0xcc, 0x52, 0x6a, 0x54, 0x86, 0x9c, 0x38, 0x85, 0x92, 0x7e, 0xa6,
	0x83, 0xcb, 0x16, 0xf2, 0x12, 0xd6, 0xfa, 0xf8, 0x95, 0x3b, 0x43, 0x74, 0xe6, 0xc4, 0xcc, 0xd5,
	0x34, 0x7c, 0xae, 0x28, 0x19, 0x3f, 0x34, 0x20, 0x17, 0x51, 0xeb, 0xfe, 0xac, 0x42, 0xca, 0xb0,
	0x92, 0x08, 0x40, 0xe1, 0x6b, 0xb9, 0xb4, 0x82, 0x02, 0x50, 0xd6, 0x37, 0x3f, 0xa4, 0xd6, 0x3f,
	0x71, 0x59, 0xd7, 0x86, 0xac, 0x3c, 0x7d, 0x36, 0xbe, 0x6b, 0xa0, 0x8b, 0xab, 0x38, 0xc4, 0xe9,
	0x66, 0x6f, 0x2d, 0xd8, 0x90, 0xc8, 0x99, 0x0f, 0xe2, 0x2c, 0xae, 0xf4, 0xd9, 0x9d, 0x4e, 0xf2,
	0xee, 0x29, 0xed, 0xc7, 0xc9, 0x9d, 0x18, 0x33, 0x9a, 0xb0, 0x35, 0x81, 0xc1, 0xfd, 0xfa, 0xc1,
	0x02, 0x52, 0xc3, 0x10, 0xc7, 0x44, 0xff, 0xf3, 0x17, 0xc5, 0x78, 0x27, 0x65, 0x19, 0xea, 0x9a,
	0xe9, 0xbb, 0x60, 0x1c, 0xc9, 0xd3, 0x8c, 0x36, 0x4e, 0x70, 0xe1, 0xfc, 0x74, 0x17, 0xee, 0xec,
	0x42, 0xfe, 0xc6, 0x0a, 0x24, 0x0f, 0x8b, 0x95, 0x83, 0xc6, 0x71, 0x75, 0xfd, 0x01, 0xd1, 0xe1,
	0xc9, 0xf1, 0x69, 0xb5, 0x7e, 0x51, 0x3b, 0x74, 0x6a, 0x67, 0x9f, 0x4f, 0xeb, 0x67, 0x07, 0x35,
	0xe7, 0xc2, 0xae, 0xaf, 0x6b, 0x95, 0x9f, 0x1a, 0x94, 0x3c, 0xda, 0x53, 0x98, 0x3e, 0x52, 0x33,
	0xf1, 0xbd, 0xc9, 0x42, 0x55, 0xb6, 0x1b, 0x22, 0x9c, 0xa2, 0x9f, 0x67, 0x51, 0x3b, 0xf2, 0x4e,
	0xe4, 0xcf, 0xe5, 0x4b, 0x55, 0x61, 0xd0, 0xd0, 0xed, 0xfb, 0x26, 0x8d, 0x7d, 0xcb, 0xc7, 0xbe,
	0xb8, 0x4b, 0x56, 0x96, 0x72, 0xa3, 0x80, 0x4d, 0xfe, 0x39, 0x94, 0xe5, 0x63, 0x33, 0x27, 0xea,
	0xdf, 0xfe, 0x0e, 0x00, 0x00, 0xff, 0xff, 0x5c, 0x7c, 0xa0, 0x45, 0xd4, 0x06, 0x00, 0x00,
}
