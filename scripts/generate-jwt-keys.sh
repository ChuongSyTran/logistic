#!/usr/bin/env bash
#
# Sinh cặp khoá RSA dùng để ký và xác thực JWT.
#
#   auth_service  giữ jwt_private.pem  -> KÝ token
#   gateway       giữ jwt_public.pem   -> VERIFY token
#
# Tách hai vai trò như vậy là điểm mấu chốt: gateway phơi ra Internet và là thứ
# dễ bị chiếm nhất; nó verify được nhưng KHÔNG ký được. Với HS256 (bản trước),
# hai bên dùng chung một secret, nên chiếm được gateway là phát hành được token
# admin.
#
# Dùng:
#   ./scripts/generate-jwt-keys.sh            # sinh vào ./secrets
#   ./scripts/generate-jwt-keys.sh /đường/dẫn # sinh vào chỗ khác
#
set -euo pipefail

OUT_DIR="${1:-secrets}"
PRIVATE_KEY="$OUT_DIR/jwt_private.pem"
PUBLIC_KEY="$OUT_DIR/jwt_public.pem"

if ! command -v openssl >/dev/null 2>&1; then
  echo "Cần openssl. Trên Windows thì Git Bash đã có sẵn." >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

# Không ghi đè khoá đang dùng: làm vậy sẽ khiến MỌI token đang lưu hành thành vô
# hiệu ngay lập tức, và mọi người dùng bị đăng xuất mà không rõ vì sao.
if [ -f "$PRIVATE_KEY" ]; then
  echo "$PRIVATE_KEY đã tồn tại."
  echo "Muốn xoay khoá thì làm theo docs/flows/authentication-flow.md — đừng ghi đè."
  exit 1
fi

# 2048 bit là mức tối thiểu NIST SP 800-131A còn chấp nhận. 4096 an toàn hơn về
# lâu dài nhưng verify chậm hơn khoảng ba lần, mà verify chạy ở MỌI request.
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$PRIVATE_KEY" 2>/dev/null
openssl rsa -in "$PRIVATE_KEY" -pubout -out "$PUBLIC_KEY" 2>/dev/null

chmod 600 "$PRIVATE_KEY" 2>/dev/null || true
chmod 644 "$PUBLIC_KEY" 2>/dev/null || true

echo "Đã sinh cặp khoá:"
echo "  private (chỉ auth_service): $PRIVATE_KEY"
echo "  public  (gateway)         : $PUBLIC_KEY"
echo
echo "Trong .env, hai biến này đã trỏ sẵn tới đường dẫn mà docker-compose mount:"
echo "  AUTH_SERVICE_JWT_PRIVATE_KEY=/run/secrets/jwt_private.pem"
echo "  GATEWAY_JWT_PUBLIC_KEY=/run/secrets/jwt_public.pem"
echo
echo "Chạy ngoài Docker thì trỏ thẳng vào file vừa sinh:"
echo "  AUTH_SERVICE_JWT_PRIVATE_KEY=$PRIVATE_KEY"
echo "  GATEWAY_JWT_PUBLIC_KEY=$PUBLIC_KEY"
echo
echo "KHÔNG commit private key. Thư mục secrets/ đã nằm trong .gitignore."
