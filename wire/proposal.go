package wire

import (
	"bytes"
	"io"
	"reflect"

	"github.com/massnetorg/MassNet-wallet/errors"
	"github.com/massnetorg/MassNet-wallet/btcec"
	wirepb "github.com/massnetorg/MassNet-wallet/wire/pb"

	"github.com/golang/protobuf/proto"
)

const (
	typeFaultPubKey = iota
	typePlaceHolder
	typeAnyMessage
)

const (
	HeadersPerProposal = 2
	//versionBytes       = 4
	//typeBytes          = 4
)

const ProposalVersion = 1

type AnyMessage struct {
	Data   []byte
	Length uint16
}

func writeAnyMessage(w io.Writer, am *AnyMessage, mode CodecMode) error {
	buf := am.Data
	_, err := w.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func readAnyMessage(r io.Reader, am *AnyMessage, mode CodecMode) error {
	err := readElement(r, &am.Length)
	if err != nil {
		return err
	}
	buf := make([]byte, am.Length, am.Length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	am.Data = buf
	return nil
}

func (am *AnyMessage) Serialize(w io.Writer, mode CodecMode) error {
	return writeAnyMessage(w, am, mode)
}

func (am *AnyMessage) Deserialize(r io.Reader, mode CodecMode) error {
	return readAnyMessage(r, am, mode)
}

type DefaultMessage struct {
	Data   []byte
	Type   int32
	Length uint16
}

func writeDefaultMessage(w io.Writer, dm *DefaultMessage, mode CodecMode) error {
	buf := dm.Data
	_, err := w.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func readDefaultMessage(r io.Reader, dm *DefaultMessage, mode CodecMode) error {
	err := readElement(r, &dm.Length)
	if err != nil {
		return err
	}
	buf := make([]byte, dm.Length, dm.Length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	dm.Data = buf
	return nil
}

func (dm *DefaultMessage) Serialize(w io.Writer, mode CodecMode) error {

	return writeDefaultMessage(w, dm, mode)
}

func (dm *DefaultMessage) Deserialize(r io.Reader, mode CodecMode) error {

	return readDefaultMessage(r, dm, mode)
}

type FaultPubKey struct {
	Pk        *btcec.PublicKey
	Testimony [HeadersPerProposal]*BlockHeader
}

var contentType map[int32]reflect.Type

func init() {
	contentType = make(map[int32]reflect.Type)
	contentType[typeAnyMessage] = reflect.TypeOf([]byte{})
	contentType[typeFaultPubKey] = reflect.TypeOf(FaultPubKey{})
}

func writeFaultPubKey(w io.Writer, fpk *FaultPubKey, mode CodecMode) error {
	for i := 0; i < HeadersPerProposal; i++ {
		err := writeBlockHeader(w, fpk.Testimony[i], mode)
		if err != nil {
			return err
		}
	}
	return nil
}

func readFaultPubKey(r io.Reader, fpk *FaultPubKey, mode CodecMode) error {
	for i := 0; i < HeadersPerProposal; i++ {
		bh := NewEmptyBlockHeader()
		err := readBlockHeader(r, bh, mode)
		fpk.Testimony[i] = bh
		if err != nil {
			return err
		}
	}
	fpk.Pk = fpk.Testimony[HeadersPerProposal-1].PubKey
	return nil
}

func NewEmptyFaultPubKey() *FaultPubKey {
	var t [HeadersPerProposal]*BlockHeader
	for i := 0; i < HeadersPerProposal; i++ {
		bh := NewEmptyBlockHeader()
		t[i] = bh
	}
	return &FaultPubKey{
		Pk:        new(btcec.PublicKey),
		Testimony: t,
	}
}

func NewFaultPubKeyFromBytes(b []byte) (*FaultPubKey, error) {
	pb := new(wirepb.Punishment)

	err := proto.Unmarshal(b, pb)
	if err != nil {
		return nil, err
	}

	h1 := NewBlockHeaderFromProto(pb.TestimonyA)
	h2 := NewBlockHeaderFromProto(pb.TestimonyB)
	var testimony [2]*BlockHeader
	testimony[0] = h1
	testimony[1] = h2
	pk := h1.PubKey
	if !reflect.DeepEqual(pk, h2.PubKey) {
		return nil, errors.New("invalid Punishment Proposal, different PublicKey")
	}

	fpk := &FaultPubKey{
		Pk:        pk,
		Testimony: testimony,
	}

	return fpk, nil
}

func (fpk *FaultPubKey) Serialize(w io.Writer, mode CodecMode) error {

	return writeFaultPubKey(w, fpk, mode)
}

func (fpk *FaultPubKey) Deserialize(r io.Reader, mode CodecMode) error {

	return readFaultPubKey(r, fpk, mode)
}

func (fpk *FaultPubKey) Bytes() ([]byte, error) {
	pb := &wirepb.Punishment{
		Version:    ProposalVersion,
		Type:       typeFaultPubKey,
		TestimonyA: fpk.Testimony[0].ToProto(),
		TestimonyB: fpk.Testimony[1].ToProto(),
	}

	buf, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

type FaultPubKeyList []*FaultPubKey

func (fpkList FaultPubKeyList) Len() int {
	return len(fpkList)
}

type PlaceHolder struct {
	Data []byte
}

func NewPlaceHolder() *PlaceHolder {
	buf := make([]byte, 2*blockHeaderMinLen, 2*blockHeaderMinLen)
	return &PlaceHolder{
		Data: buf,
	}
}

func writePlaceHolder(w io.Writer, ph *PlaceHolder, mode CodecMode) error {
	_, err := w.Write(ph.Data[:])
	if err != nil {
		return err
	}
	return nil
}

func readPlaceHolder(r io.Reader, ph *PlaceHolder, mode CodecMode) error {
	//var buf [2 * blockHeaderMinLen]byte{}
	buf := make([]byte, 2*blockHeaderMinLen, 2*blockHeaderMinLen)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	ph.Data = buf
	return nil
}

func (ph *PlaceHolder) Serialize(w io.Writer, mode CodecMode) error {
	return writePlaceHolder(w, ph, mode)
}

func (ph *PlaceHolder) Deserialize(r io.Reader, mode CodecMode) error {
	return readPlaceHolder(r, ph, mode)
}

type proposalContent interface {
	Serialize(w io.Writer, mode CodecMode) error
	Deserialize(r io.Reader, mode CodecMode) error
}

type Proposal struct {
	version      int32
	proposalType int32
	content      proposalContent
}

// structure of Punishment Proposal
//
//   ------------------------------------------------------------------
//  | version | proposalType |                 content                 |
//   ------------------------------------------------------------------
//  |  int32  |    int32     |    BlockHeader1    |    BlockHeader2    |
//   ------------------------------------------------------------------
//  | 4 Bytes |   4 Bytes    | BlockHeaderLength1 | BlockHeaderLength2 |
//   ------------------------------------------------------------------
//
// structure of default Proposal and PlaceHolder
//
//   -------------------------------------
//  | version | proposalType |  content   |
//   -------------------------------------
//  |  int32  |    int32     |    Data    |
//   -------------------------------------
//  | 4 Bytes |   4 Bytes    | DataLength |
//   -------------------------------------
//

func NewProposalFromAnyMessage(am *AnyMessage) *Proposal {
	return &Proposal{
		version:      ProposalVersion,
		proposalType: typeAnyMessage,
		content:      am,
	}
}

func NewProposalFromFaultPubKey(fpk *FaultPubKey) *Proposal {
	return &Proposal{
		version:      ProposalVersion,
		proposalType: typeFaultPubKey,
		content:      fpk,
	}
}

func NewProposalFromPlaceHolder(ph *PlaceHolder) *Proposal {
	return &Proposal{
		version:      ProposalVersion,
		proposalType: typePlaceHolder,
		content:      ph,
	}
}

func NewProposalFromDefaultMessage(dm *DefaultMessage) *Proposal {
	return &Proposal{
		version:      ProposalVersion,
		proposalType: dm.Type,
		content:      dm,
	}
}

func NewFaultPubKeyFromProposal(p *Proposal) (*FaultPubKey, error) {
	fpk := NewEmptyFaultPubKey()
	if p.proposalType != typeFaultPubKey {
		return fpk, errors.New("wrong type in PunishmentArea")
	}
	punishment, ok := p.content.(*FaultPubKey)
	if !ok {
		return fpk, errors.New("wrong type in content")
	}
	fpk = punishment
	return fpk, nil
}

func GetFaultPubKeyListFromProposalList(proposals []*Proposal) ([]*FaultPubKey, error) {
	faultPubKeyList := make([]*FaultPubKey, len(proposals))
	for i, proposal := range proposals {
		//fpk := NewEmptyFaultPubKey()
		fpk, err := NewFaultPubKeyFromProposal(proposal)
		if err != nil {
			return faultPubKeyList, err
		}
		faultPubKeyList[i] = fpk
	}
	return faultPubKeyList, nil
}

func (p *Proposal) Version() int32 {
	return p.version
}

func (p *Proposal) Content() (interface{}, reflect.Type) {
	return p.content, contentType[p.proposalType]
}

func (p *Proposal) Type() (int32, reflect.Type) {
	return p.proposalType, contentType[p.proposalType]
}

func (p *Proposal) Serialize(w io.Writer, mode CodecMode) error {

	return writeProposal(w, p, mode)
}

func (p *Proposal) Deserialize(r io.Reader, mode CodecMode) error {

	return readProposal(r, p, mode)
}

func (p *Proposal) Bytes(mode CodecMode) ([]byte, error) {
	var buf bytes.Buffer
	err := p.Serialize(&buf, mode)
	if err != nil {
		return nil, err
	}
	serializedProposal := buf.Bytes()
	return serializedProposal, nil
}

func (p *Proposal) Hash() Hash {
	proposalBytes, err := p.Bytes(ID)
	if err != nil {
		return Hash{}
	}
	return DoubleHashH(proposalBytes)
}

func (p *Proposal) SerializeSize() int {
	serializedProposal, _ := p.Bytes(Plain)
	return int(len(serializedProposal))
}

func writeProposal(w io.Writer, p *Proposal, mode CodecMode) error {
	switch e := p.content.(type) {
	case *AnyMessage:
		return writeAnyMessage(w, e, mode)
	case *FaultPubKey:
		return writeFaultPubKey(w, e, mode)
	case *PlaceHolder:
		return writePlaceHolder(w, e, mode)
	case *DefaultMessage:
		return writeDefaultMessage(w, e, mode)
	}
	return nil
}

func readProposal(r io.Reader, p *Proposal, mode CodecMode) error {
	err := readElements(r, &p.version, &p.proposalType)
	if err != nil {
		return err
	}

	switch e := p.proposalType; e {
	case typeAnyMessage:
		am := &AnyMessage{}
		err := readAnyMessage(r, am, mode)
		if err != nil {
			return err
		}
		p.content = am
		return nil

	case typeFaultPubKey:
		fpk := NewEmptyFaultPubKey()
		err := readFaultPubKey(r, fpk, mode)
		if err != nil {
			return err
		}
		p.content = fpk
		return nil

	case typePlaceHolder:
		ph := &PlaceHolder{}
		err := readPlaceHolder(r, ph, mode)
		if err != nil {
			return err
		}
		p.content = ph
		return nil

	default:
		dm := &DefaultMessage{}
		err := readDefaultMessage(r, dm, mode)
		if err != nil {
			return err
		}
		p.content = dm
		return nil
	}
}

type ProposalArea struct {
	AllCount        uint16
	PunishmentCount uint16
	PunishmentArea  []*Proposal
	OtherArea       []*Proposal
}

// structure of ProposalArea
//
//   ---------------------------------------------------------------------------
//  | AllCount  | PunishmentCount |    PunishmentArea    |      OtherArea       |
//   ---------------------------------------------------------------------------
//  |   uint16  |     uint16      |    Proposal    | ... |    Proposal    | ... |
//  |---------------------------------------------------------------------------
//  |  2 Bytes  |     2 Bytes     | ProposalLength | ... | ProposalLength | ... |
//   ---------------------------------------------------------------------------
//

func newEmptyProposalArea() *ProposalArea {
	return &ProposalArea{
		AllCount:        uint16(0),
		PunishmentCount: uint16(0),
		PunishmentArea:  []*Proposal{},
		OtherArea:       []*Proposal{},
	}
}

func NewProposalAreaFromProposalList(punishmentArea []*Proposal, otherArea []*Proposal) (*ProposalArea, error) {
	for _, punishment := range punishmentArea {
		if punishment.proposalType != typeFaultPubKey {
			return nil, errors.New("wrong type in PunishmentArea")
		}
	}
	punishmentCount := len(punishmentArea)

	for _, other := range otherArea {
		if other.proposalType == typeFaultPubKey {
			return nil, errors.New("wrong type in OtherArea")
		}
	}

	return &ProposalArea{
		AllCount:        uint16(punishmentCount + len(otherArea)),
		PunishmentCount: uint16(punishmentCount),
		PunishmentArea:  punishmentArea,
		OtherArea:       otherArea,
	}, nil
}

func writeProposalArea(w io.Writer, pa *ProposalArea, mode CodecMode) error {
	pb, err := pa.ToProto()
	if err != nil {
		return err
	}

	content, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

func readProposalArea(r io.Reader, pa *ProposalArea, mode CodecMode) error {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}

	pb := new(wirepb.ProposalArea)
	err = proto.Unmarshal(buf.Bytes(), pb)
	if err != nil {
		return err
	}

	err = pa.FromProto(pb)
	return err
}

func (pa *ProposalArea) Serialize(w io.Writer, mode CodecMode) error {
	return writeProposalArea(w, pa, mode)
}

func (pa *ProposalArea) Deserialize(r io.Reader, mode CodecMode) error {
	return readProposalArea(r, pa, mode)
}

func (pa *ProposalArea) GetAllProposals() []*Proposal {
	allProposal := make([]*Proposal, 0, pa.AllCount)
	for _, p := range pa.PunishmentArea {
		allProposal = append(allProposal, p)
	}
	for _, p := range pa.OtherArea {
		allProposal = append(allProposal, p)
	}
	return allProposal
}

// ToProto get proto ProposalArea from wire ProposalArea
func (pa *ProposalArea) ToProto() (*wirepb.ProposalArea, error) {
	if len(pa.PunishmentArea)+len(pa.OtherArea) != int(pa.AllCount) {
		// FIXME: error msg?
		return nil, errors.New("number of Proposals is incorrect")
	}

	if len(pa.PunishmentArea) != int(pa.PunishmentCount) {
		// FIXME: error msg?
		return nil, errors.New("number of PunishmentProposals is incorrect")
	}

	punishments := make([]*wirepb.Punishment, pa.PunishmentCount, pa.PunishmentCount)
	for i, proposal := range pa.PunishmentArea {
		if proposal.proposalType != typeFaultPubKey {
			return nil, errors.New("wrong type of proposal on faultPubKey")
		}
		v := proposal.content.(*FaultPubKey)
		p := &wirepb.Punishment{
			Version:    proposal.version,
			Type:       proposal.proposalType,
			TestimonyA: v.Testimony[0].ToProto(),
			TestimonyB: v.Testimony[1].ToProto(),
		}
		punishments[i] = p
	}

	placeHolder := &wirepb.Proposal{
		Version: ProposalVersion,
		Type:    typePlaceHolder,
	}
	if pa.PunishmentCount > 0 {
		placeHolder.Content = make([]byte, 0, 0)
	} else {
		placeHolder.Content = NewPlaceHolder().Data
	}

	others := make([]*wirepb.Proposal, len(pa.OtherArea), len(pa.OtherArea))
	for i, proposal := range pa.OtherArea {
		if proposal.proposalType < typeAnyMessage {
			return nil, errors.New("wrong type of proposal on otherArea")
		}
		s, err := proposal.Bytes(Plain)
		if err != nil {
			return nil, err
		}
		p := &wirepb.Proposal{
			Version: proposal.version,
			Type:    proposal.proposalType,
			Content: s,
		}
		others[i] = p
	}

	return &wirepb.ProposalArea{
		Punishments:    punishments,
		PlaceHolder:    placeHolder,
		OtherProposals: others,
	}, nil
}

// FromProto load proto ProposalArea into wire ProposalArea,
// if error happens, old content is still immutable
func (pa *ProposalArea) FromProto(pb *wirepb.ProposalArea) error {
	if len(pb.Punishments) == 0 && len(pb.PlaceHolder.Content) != len(NewPlaceHolder().Data) {
		return errors.New("invalid placeHolder for non-Punishment ProposalArea")
	}
	punishments := make([]*Proposal, len(pb.Punishments), len(pb.Punishments))
	for i, v := range pb.Punishments {
		h1 := NewBlockHeaderFromProto(v.TestimonyA)
		h2 := NewBlockHeaderFromProto(v.TestimonyB)
		var testimony [2]*BlockHeader
		testimony[0] = h1
		testimony[1] = h2
		pk := h1.PubKey
		if !reflect.DeepEqual(pk, h2.PubKey) {
			return errors.New("invalid Punishment Proposal, different PublicKey")
		}
		p := &Proposal{
			version:      v.Version,
			proposalType: v.Type,
			content: &FaultPubKey{
				Pk:        pk,
				Testimony: testimony,
			},
		}
		punishments[i] = p
	}

	others := make([]*Proposal, len(pb.OtherProposals), len(pb.OtherProposals))
	for i, v := range pb.OtherProposals {
		if v.Type < typeAnyMessage {
			return errors.New("wrong type of proposal on proto otherArea")
		}
		var content proposalContent
		if v.Type == typeAnyMessage {
			content = &AnyMessage{
				Data:   v.Content,
				Length: uint16(len(v.Content)),
			}
		} else {
			content = &DefaultMessage{
				Data:   v.Content,
				Length: uint16(len(v.Content)),
			}
		}
		p := &Proposal{
			version:      v.Version,
			proposalType: v.Type,
			content:      content,
		}
		others[i] = p
	}

	pa.AllCount = uint16(len(punishments) + len(others))
	pa.PunishmentCount = uint16(len(punishments))
	pa.PunishmentArea = punishments
	pa.OtherArea = others
	return nil
}

// NewProposalAreaFromProto get wire ProposalArea from proto ProposalArea
func NewProposalAreaFromProto(pb *wirepb.ProposalArea) (*ProposalArea, error) {
	pa := new(ProposalArea)
	err := pa.FromProto(pb)
	if err != nil {
		return nil, err
	}
	return pa, nil
}
