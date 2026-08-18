package converters

import (
	"database/sql/driver"
	"encoding/binary"
	"math"
	"strconv"

	"github.com/oracle/go-oracledb/v26/internal/common"
	driverCommon "github.com/oracle/go-oracledb/v26/internal/driver/common"
	oracleErrors "github.com/oracle/go-oracledb/v26/oracle/errors"
)

const (
	vectorTypeName            = "VECTOR"
	maxVectorDimensions       = 65535
	maxVectorBinaryDimensions = 65528

	vectorMagic           = 0xDB
	vectorVersionV0       = 0
	vectorVersionV2       = 2
	vectorBaseHeaderBytes = 9
	vectorNormBytes       = 8
	vectorHeaderBytes     = vectorBaseHeaderBytes + vectorNormBytes

	vectorFlagOptional   = 0x8000
	vectorFlagLittle     = 0x0001
	vectorFlagNorm       = 0x0002
	vectorFlagIEEE       = 0x0008
	vectorFlagNormSource = 0x0010
	vectorFlagSparse     = 0x0020

	vectorTypeFloat32 = 0x02
	vectorTypeFloat64 = 0x03
	vectorTypeInt8    = 0x04
	vectorTypeBinary  = 0x05
)

// EncodeVectorFloat64 encodes common.VectorFloat64 into Oracle VECTOR(FLOAT64) wire layout.
func EncodeVectorFloat64(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.VectorFloat64)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.VectorFloat64")
	}
	if values == nil {
		return nil, nil
	}
	return encodeVectorFloat64([]float64(values))
}

// EncodeVectorFloat32 encodes common.VectorFloat32 into Oracle VECTOR(FLOAT32) wire layout.
func EncodeVectorFloat32(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.VectorFloat32)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.VectorFloat32")
	}
	if values == nil {
		return nil, nil
	}
	return encodeVectorFloat32([]float32(values))
}

// EncodeVectorInt8 encodes common.VectorInt8 into Oracle VECTOR(INT8) wire layout.
func EncodeVectorInt8(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.VectorInt8)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.VectorInt8")
	}
	if values == nil {
		return nil, nil
	}
	return encodeVectorInt8([]int8(values))
}

// EncodeVectorBinary encodes packed bits into Oracle VECTOR(BINARY) wire layout.
func EncodeVectorBinary(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.VectorBinary)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.VectorBinary")
	}
	if values == nil {
		return nil, nil
	}
	return encodeVectorBinary(values)
}

// EncodeSparseVectorFloat64 encodes common.SparseVectorFloat64 into Oracle
// SPARSE VECTOR(FLOAT64) wire layout.
func EncodeSparseVectorFloat64(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.SparseVectorFloat64)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.SparseVectorFloat64")
	}
	if err := validateSparseVector(values.Dimensions, values.Indices, len(values.Values)); err != nil {
		return nil, err
	}
	return encodeSparseVectorFloat64(values)
}

// EncodeSparseVectorFloat32 encodes common.SparseVectorFloat32 into Oracle
// SPARSE VECTOR(FLOAT32) wire layout.
func EncodeSparseVectorFloat32(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.SparseVectorFloat32)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.SparseVectorFloat32")
	}
	if err := validateSparseVector(values.Dimensions, values.Indices, len(values.Values)); err != nil {
		return nil, err
	}
	return encodeSparseVectorFloat32(values)
}

// EncodeSparseVectorInt8 encodes common.SparseVectorInt8 into Oracle SPARSE
// VECTOR(INT8) wire layout.
func EncodeSparseVectorInt8(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.SparseVectorInt8)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.SparseVectorInt8")
	}
	if err := validateSparseVector(values.Dimensions, values.Indices, len(values.Values)); err != nil {
		return nil, err
	}
	return encodeSparseVectorInt8(values)
}

// EncodeSparseVectorBinary encodes common.SparseVectorBinary into Oracle
// SPARSE VECTOR(BINARY) wire layout.
func EncodeSparseVectorBinary(v driver.Value) (driverCommon.B1Array, error) {
	values, ok := v.(driverCommon.SparseVectorBinary)
	if !ok {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidValue, "common.SparseVectorBinary")
	}
	if err := validateSparseVector(values.Dimensions, values.Indices, len(values.Indices)); err != nil {
		return nil, err
	}
	return encodeSparseVectorBinary(values)
}

// DecodeVector decodes Oracle VECTOR data into the matching Go slice type:
// []float64, []float32, []int8, or []byte (packed bits for BINARY vectors).
func DecodeVector(data driverCommon.B1Array) (driver.Value, error) {
	if len(data) == 0 {
		return nil, nil
	}
	return decodeVector(data)
}

func encodeVectorFloat64(values []float64) (driverCommon.B1Array, error) {
	if len(values) > maxVectorDimensions {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonOutOfRange, "dimensions<=65535")
	}
	payloadBytes := len(values) * 8
	out := make([]byte, vectorHeaderBytes+payloadBytes)
	writeVectorHeader(out, vectorVersionV0, vectorFlagNorm|vectorFlagNormSource, vectorTypeFloat64, uint32(len(values)))
	offset := vectorHeaderBytes
	for _, value := range values {
		putOracleBinaryDouble(out[offset:offset+8], value)
		offset += 8
	}
	return out, nil
}

func encodeVectorFloat32(values []float32) (driverCommon.B1Array, error) {
	if len(values) > maxVectorDimensions {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonOutOfRange, "dimensions<=65535")
	}
	payloadBytes := len(values) * 4
	out := make([]byte, vectorHeaderBytes+payloadBytes)
	writeVectorHeader(out, vectorVersionV0, vectorFlagNorm|vectorFlagNormSource, vectorTypeFloat32, uint32(len(values)))
	offset := vectorHeaderBytes
	for _, value := range values {
		putOracleBinaryFloat(out[offset:offset+4], value)
		offset += 4
	}
	return out, nil
}

func encodeVectorInt8(values []int8) (driverCommon.B1Array, error) {
	if len(values) > maxVectorDimensions {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonOutOfRange, "dimensions<=65535")
	}
	out := make([]byte, vectorHeaderBytes+len(values))
	writeVectorHeader(out, vectorVersionV0, vectorFlagNorm|vectorFlagNormSource, vectorTypeInt8, uint32(len(values)))
	for i := range values {
		out[vectorHeaderBytes+i] = byte(values[i])
	}
	return out, nil
}

func encodeVectorBinary(values driverCommon.VectorBinary) (driverCommon.B1Array, error) {
	if len(values) > maxVectorBinaryDimensions/8 {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonOutOfRange, "dimensions<=65528")
	}
	out := make([]byte, vectorHeaderBytes+len(values))
	writeVectorHeader(out, vectorVersionV0, vectorFlagNormSource, vectorTypeBinary, uint32(len(values))*8)
	copy(out[vectorHeaderBytes:], values)
	return out, nil
}

func encodeSparseVectorFloat64(values driverCommon.SparseVectorFloat64) (driverCommon.B1Array, error) {
	out := makeSparseVectorBuffer(vectorTypeFloat64, values.Dimensions, len(values.Indices), len(values.Values)*8, vectorFlagNorm|vectorFlagNormSource)
	offset := writeSparseVectorIndices(out, values.Indices)
	for _, value := range values.Values {
		putOracleBinaryDouble(out[offset:offset+8], value)
		offset += 8
	}
	return out, nil
}

func encodeSparseVectorFloat32(values driverCommon.SparseVectorFloat32) (driverCommon.B1Array, error) {
	out := makeSparseVectorBuffer(vectorTypeFloat32, values.Dimensions, len(values.Indices), len(values.Values)*4, vectorFlagNorm|vectorFlagNormSource)
	offset := writeSparseVectorIndices(out, values.Indices)
	for _, value := range values.Values {
		putOracleBinaryFloat(out[offset:offset+4], value)
		offset += 4
	}
	return out, nil
}

func encodeSparseVectorInt8(values driverCommon.SparseVectorInt8) (driverCommon.B1Array, error) {
	out := makeSparseVectorBuffer(vectorTypeInt8, values.Dimensions, len(values.Indices), len(values.Values), vectorFlagNorm|vectorFlagNormSource)
	offset := writeSparseVectorIndices(out, values.Indices)
	for _, value := range values.Values {
		out[offset] = byte(value)
		offset++
	}
	return out, nil
}

func encodeSparseVectorBinary(values driverCommon.SparseVectorBinary) (driverCommon.B1Array, error) {
	out := makeSparseVectorBuffer(vectorTypeBinary, values.Dimensions, len(values.Indices), 0, vectorFlagNormSource)
	writeSparseVectorIndices(out, values.Indices)
	return out, nil
}

func makeSparseVectorBuffer(typeCode byte, dimensions uint32, count, valueBytes int, flags uint16) driverCommon.B1Array {
	out := make(driverCommon.B1Array, vectorHeaderBytes+2+count*4+valueBytes)
	writeVectorHeader(out, vectorVersionV2, vectorFlagSparse|flags, typeCode, dimensions)
	return out
}

func writeSparseVectorIndices(out driverCommon.B1Array, indices []uint32) int {
	offset := vectorHeaderBytes
	binary.BigEndian.PutUint16(out[offset:offset+2], uint16(len(indices)))
	offset += 2
	for _, index := range indices {
		binary.BigEndian.PutUint32(out[offset:offset+4], index)
		offset += 4
	}
	return offset
}

func validateSparseVector(dimensions uint32, indices []uint32, valuesLength int) error {
	if dimensions > maxVectorDimensions {
		return common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonOutOfRange, "dimensions<=65535")
	}
	if len(indices) > maxVectorDimensions {
		return common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonOutOfRange, "non-zero-dimensions<=65535")
	}
	if len(indices) != valuesLength {
		return common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Encode", driverCommon.ReasonInvalidLength, "indices==values")
	}
	return nil
}

func writeVectorHeader(out []byte, version byte, flags uint16, typeCode byte, dimensions uint32) {
	out[0] = vectorMagic
	out[1] = version
	binary.BigEndian.PutUint16(out[2:4], flags)
	out[4] = typeCode
	binary.BigEndian.PutUint32(out[5:9], dimensions)
	// TTC clients reserve this FLOAT64-sized field. For numeric vectors,
	// NORM|NORMSRC asks the database to compute the norm; BINARY reserves the
	// field with NORMSRC because a BINARY vector has no norm. A zero value is
	// therefore intentional, not an encoded norm calculated by this driver.
	clear(out[vectorBaseHeaderBytes:vectorHeaderBytes])
}

func decodeVector(data []byte) (driver.Value, error) {
	if len(data) < vectorBaseHeaderBytes {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidLength, ">="+strconv.Itoa(vectorBaseHeaderBytes))
	}
	if data[0] != vectorMagic {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidFormat, "magic=0xDB")
	}

	version := data[1]
	flags := binary.BigEndian.Uint16(data[2:4])
	typeCode := data[4]
	dimensionCount := binary.BigEndian.Uint32(data[5:9])

	if flags&vectorFlagOptional != 0 {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidFormat, "optional-flags-unsupported")
	}
	payloadOffset := vectorBaseHeaderBytes
	if flags&(vectorFlagNorm|vectorFlagNormSource) != 0 {
		// The norm field is present whenever either flag is set. Accepting both
		// forms keeps decoding compatible with clients that omit norm metadata.
		payloadOffset += vectorNormBytes
	}
	if flags&vectorFlagSparse != 0 {
		if version != vectorVersionV2 {
			return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidValue, "sparse-version=2")
		}
		return decodeSparseVector(data[payloadOffset:], typeCode, dimensionCount, flags)
	}
	if version != vectorVersionV0 {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidValue, "version=0")
	}

	valueBytes, err := vectorValueByteLength(typeCode, dimensionCount)
	if err != nil {
		return nil, err
	}
	expected := payloadOffset + valueBytes
	if len(data) != expected {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidLength, "=="+strconv.Itoa(expected))
	}

	payload := data[payloadOffset:]
	switch typeCode {
	case vectorTypeFloat64:
		out := make([]float64, dimensionCount)
		for i := range out {
			start := i * 8
			value, err := decodeFloat64Dimension(payload[start:start+8], flags)
			if err != nil {
				return nil, err
			}
			out[i] = value
		}
		return out, nil
	case vectorTypeFloat32:
		out := make([]float32, dimensionCount)
		for i := range out {
			start := i * 4
			value, err := decodeFloat32Dimension(payload[start:start+4], flags)
			if err != nil {
				return nil, err
			}
			out[i] = value
		}
		return out, nil
	case vectorTypeInt8:
		out := make([]int8, dimensionCount)
		for i := range out {
			out[i] = int8(payload[i])
		}
		return out, nil
	case vectorTypeBinary:
		out := make([]byte, len(payload))
		copy(out, payload)
		return out, nil
	default:
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidFormat, "type=2|3|4|5")
	}
}

func decodeSparseVector(payload []byte, typeCode byte, dimensions uint32, flags uint16) (driver.Value, error) {
	if dimensions > maxVectorDimensions {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonOutOfRange, "dimension-count")
	}
	if len(payload) < 2 {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidLength, ">=2")
	}
	count := int(binary.BigEndian.Uint16(payload[:2]))
	indicesBytes := count * 4
	if len(payload) < 2+indicesBytes {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidLength, "sparse-indices")
	}
	indices := make([]uint32, count)
	offset := 2
	for i := range indices {
		indices[i] = binary.BigEndian.Uint32(payload[offset : offset+4])
		offset += 4
	}

	valueBytes, err := vectorSparseValueByteLength(typeCode, count)
	if err != nil {
		return nil, err
	}
	if len(payload) != offset+valueBytes {
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidLength, "sparse-values")
	}
	values := payload[offset:]
	switch typeCode {
	case vectorTypeFloat64:
		out := driverCommon.SparseVectorFloat64{Dimensions: dimensions, Indices: indices, Values: make([]float64, count)}
		for i := range out.Values {
			value, err := decodeFloat64Dimension(values[i*8:(i+1)*8], flags)
			if err != nil {
				return nil, err
			}
			out.Values[i] = value
		}
		return out, nil
	case vectorTypeFloat32:
		out := driverCommon.SparseVectorFloat32{Dimensions: dimensions, Indices: indices, Values: make([]float32, count)}
		for i := range out.Values {
			value, err := decodeFloat32Dimension(values[i*4:(i+1)*4], flags)
			if err != nil {
				return nil, err
			}
			out.Values[i] = value
		}
		return out, nil
	case vectorTypeInt8:
		out := driverCommon.SparseVectorInt8{Dimensions: dimensions, Indices: indices, Values: make([]int8, count)}
		for i := range out.Values {
			out.Values[i] = int8(values[i])
		}
		return out, nil
	case vectorTypeBinary:
		return driverCommon.SparseVectorBinary{Dimensions: dimensions, Indices: indices}, nil
	default:
		return nil, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidFormat, "type=2|3|4|5")
	}
}

func vectorSparseValueByteLength(typeCode byte, count int) (int, error) {
	switch typeCode {
	case vectorTypeFloat64:
		return count * 8, nil
	case vectorTypeFloat32:
		return count * 4, nil
	case vectorTypeInt8:
		return count, nil
	case vectorTypeBinary:
		return 0, nil
	default:
		return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidFormat, "unknown-type")
	}
}

func vectorValueByteLength(typeCode byte, dimensions uint32) (int, error) {
	switch typeCode {
	case vectorTypeFloat64:
		if dimensions > maxVectorDimensions {
			return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonOutOfRange, "dimension-count")
		}
		return int(dimensions) * 8, nil
	case vectorTypeFloat32:
		if dimensions > maxVectorDimensions {
			return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonOutOfRange, "dimension-count")
		}
		return int(dimensions) * 4, nil
	case vectorTypeInt8:
		if dimensions > maxVectorDimensions {
			return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonOutOfRange, "dimension-count")
		}
		return int(dimensions), nil
	case vectorTypeBinary:
		if dimensions > maxVectorBinaryDimensions {
			return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonOutOfRange, "dimension-count")
		}
		if dimensions%8 != 0 {
			return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidValue, "binary-dimension-count-multiple-of-8")
		}
		return int(dimensions / 8), nil
	default:
		return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidFormat, "unknown-type")
	}
}

func decodeFloat64Dimension(raw []byte, flags uint16) (float64, error) {
	if len(raw) != 8 {
		return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidLength, 8)
	}
	ordered := orderVectorBytes(raw, flags)
	if (flags & vectorFlagIEEE) != 0 {
		return math.Float64frombits(binary.BigEndian.Uint64(ordered)), nil
	}
	return DecodeBinaryDouble(ordered)
}

func decodeFloat32Dimension(raw []byte, flags uint16) (float32, error) {
	if len(raw) != 4 {
		return 0, common.NewOracleError(oracleErrors.ConverterExpectedFormat, nil, vectorTypeName, "Decode", driverCommon.ReasonInvalidLength, 4)
	}
	ordered := orderVectorBytes(raw, flags)
	if (flags & vectorFlagIEEE) != 0 {
		return math.Float32frombits(binary.BigEndian.Uint32(ordered)), nil
	}
	return DecodeBinaryFloat(ordered)
}

func orderVectorBytes(raw []byte, flags uint16) []byte {
	ordered := raw
	if (flags & vectorFlagLittle) != 0 {
		ordered = make([]byte, len(raw))
		for i := range raw {
			ordered[len(raw)-1-i] = raw[i]
		}
	}
	return ordered
}

func putOracleBinaryDouble(dst []byte, value float64) {
	bits := math.Float64bits(value)
	var stored uint64
	if math.Signbit(value) {
		stored = ^bits
	} else {
		stored = bits ^ 0x8000000000000000
	}
	binary.BigEndian.PutUint64(dst, stored)
}

func putOracleBinaryFloat(dst []byte, value float32) {
	bits := math.Float32bits(value)
	var stored uint32
	if math.Signbit(float64(value)) {
		stored = ^bits
	} else {
		stored = bits ^ 0x80000000
	}
	binary.BigEndian.PutUint32(dst, stored)
}
