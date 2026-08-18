package common

// VectorFloat64 represents dense VECTOR(FLOAT64) bind values.
type VectorFloat64 []float64

// VectorFloat32 represents dense VECTOR(FLOAT32) bind values.
type VectorFloat32 []float32

// VectorInt8 represents dense VECTOR(INT8) bind values.
type VectorInt8 []int8

// VectorBinary represents a packed BINARY VECTOR payload.
// Each byte stores 8 dimensions in MSB-first order.
type VectorBinary []byte

// SparseVectorFloat64 represents a sparse VECTOR(FLOAT64) bind value.
// Dimensions is the total vector length; Indices are zero-based positions of
// Values, which contains only the non-zero dimensions.
type SparseVectorFloat64 struct {
	Dimensions uint32
	Indices    []uint32
	Values     []float64
}

// SparseVectorFloat32 represents a sparse VECTOR(FLOAT32) bind value.
// Dimensions is the total vector length; Indices are zero-based positions of
// Values, which contains only the non-zero dimensions.
type SparseVectorFloat32 struct {
	Dimensions uint32
	Indices    []uint32
	Values     []float32
}

// SparseVectorInt8 represents a sparse VECTOR(INT8) bind value.
// Dimensions is the total vector length; Indices are zero-based positions of
// Values, which contains only the non-zero dimensions.
type SparseVectorInt8 struct {
	Dimensions uint32
	Indices    []uint32
	Values     []int8
}

// SparseVectorBinary represents a sparse VECTOR(BINARY) bind value.
// Dimensions is the total vector length and Indices are the zero-based
// positions of its set bits. Sparse BINARY values are implicitly one.
type SparseVectorBinary struct {
	Dimensions uint32
	Indices    []uint32
}
