package loop

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Cổng đã xanh từ trước và cây làm việc không đổi: không có bằng chứng nào về việc
// agent làm, nên không được dừng GREEN mà là NO_CHANGE. Cả hai cờ phải tới Result
// (rồi tới --json và report), không chỉ nằm trong journal.
func TestEngine_NoChangeCarriesBaselineAndUnchangedFlags(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{pass(), pass()}, fingerprints: []string{"fp-base"}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopNoChange, res.Reason)
	require.True(t, res.BaselinePass)
	require.True(t, res.Unchanged)
	require.Contains(t, res.Detail, "already passed before any change")
	require.Contains(t, res.Detail, "working tree did not change")
}

// Vế ngược: đỏ lúc đầu, agent đổi cây, rồi xanh — không cờ nào, Detail rỗng.
func TestEngine_GreenAfterARealChangeHasNoFlags(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("red"), pass()}, fingerprints: []string{"fp1"}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.False(t, res.BaselinePass)
	require.False(t, res.Unchanged)
	require.Empty(t, res.Detail)
}

// Không biết dấu vân tay (thư mục không phải repo git: driver trả giá trị luôn
// khác, test trả rỗng) thì không được kết luận "không đổi".
func TestEngine_UnknownFingerprintIsNotUnchanged(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("red"), pass()}}
	e := s.engine(baseSpec())
	e.Fingerprint = nil
	res, err := e.Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.False(t, res.Unchanged)
}

// Cổng xanh từ trước nhưng agent có đổi cây: vẫn GREEN, kèm cảnh báo cổng.
func TestEngine_BaselinePassWithAChangeIsGreenWithAWarning(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{pass(), pass()}, fingerprints: []string{"fp1"}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.True(t, res.BaselinePass)
	require.False(t, res.Unchanged)
	require.Equal(t, "verify already passed before any change", res.Detail)
}

// Đỏ lúc đầu, cây không đổi mà verify lại xanh (verify phụ thuộc thứ ngoài cây,
// hay chập chờn): vẫn GREEN vì baseline đỏ, nhưng phải cảnh báo cây không đổi.
func TestEngine_BaselineFailWithoutAChangeIsGreenWithAWarning(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("red"), pass()}, fingerprints: []string{"fp-base"}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.False(t, res.BaselinePass)
	require.True(t, res.Unchanged)
	require.Equal(t, "the working tree did not change", res.Detail)
}
