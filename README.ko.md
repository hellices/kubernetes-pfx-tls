# Kubernetes PFX to PEM 변환 Operator

Kubernetes Gateway API에서 PFX (PKCS#12) 인증서를 자동으로 PEM 형식으로 변환하는 Operator입니다.

## 개요

Kubernetes Gateway API를 사용하여 TLS 인증서를 구성할 때, 인증서는 PEM 형식이어야 합니다. 하지만 많은 조직에서는 Azure Key Vault와 같은 서비스에서 PFX 형식의 인증서를 사용합니다. 이 Operator는 Kubernetes Secret에 저장된 PFX 인증서를 자동으로 PEM 형식으로 변환하여 Gateway API 및 TLS 인증서가 필요한 다른 Kubernetes 리소스와 호환되도록 합니다.

## 주요 기능

- **자동 변환**: Secret이 생성되거나 업데이트될 때 PFX 인증서를 PEM 형식으로 자동 변환
- **어노테이션 기반**: 어노테이션을 사용하여 변환 동작 제어
- **비밀번호 지원**: 어노테이션 또는 다른 Secret의 비밀번호로 보호된 PFX 파일 지원
- **CA 인증서 처리**: PFX 번들에서 CA 인증서 추출 및 저장
- **Gateway API 호환**: 표준 `kubernetes.io/tls` 타입 Secret 출력

## 설치

### Operator 배포

1. 클러스터에 매니페스트 적용:

```bash
kubectl apply -f deploy/
```

다음이 생성됩니다:
- `pfx-tls-system` 네임스페이스
- 필요한 RBAC 리소스 (ServiceAccount, ClusterRole, ClusterRoleBinding)
- Operator 배포

### 소스에서 빌드

```bash
# 로컬 빌드
make build

# Docker 이미지 빌드
make docker-build

# Docker 이미지 푸시
make docker-push
```

## 사용 방법

### 기본 예제 (비밀번호 없음)

PFX 인증서가 포함된 Secret을 생성하고 변환을 활성화합니다:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
type: Opaque
data:
  pfx: <BASE64로_인코딩된_PFX_데이터>
```

Operator는 다음을 수행합니다:
1. 어노테이션 감지
2. PFX를 PEM 형식으로 변환
3. Secret을 다음으로 업데이트:
   - `tls.crt`: PEM 형식의 인증서
   - `tls.key`: PEM 형식의 개인 키
   - `ca.crt`: CA 인증서 (PFX에 있는 경우)
4. Secret 타입을 `kubernetes.io/tls`로 변경
5. `pfx-tls.kubernetes.io/converted: "true"` 어노테이션 추가

### 어노테이션에 비밀번호 포함 예제

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
    pfx-tls.kubernetes.io/password: "mypassword"
type: Opaque
data:
  pfx: <BASE64로_인코딩된_PFX_데이터>
```

### 다른 Secret에 비밀번호 포함 예제

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pfx-password
  namespace: default
type: Opaque
data:
  password: <BASE64로_인코딩된_비밀번호>
---
apiVersion: v1
kind: Secret
metadata:
  name: my-pfx-cert
  namespace: default
  annotations:
    pfx-tls.kubernetes.io/convert: "true"
    pfx-tls.kubernetes.io/password-secret-name: "pfx-password"
    pfx-tls.kubernetes.io/password-secret-key: "password"
type: Opaque
data:
  pfx: <BASE64로_인코딩된_PFX_데이터>
```

### Gateway API와 함께 사용

변환 후 Gateway API에서 Secret 사용:

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: my-gateway
  namespace: default
spec:
  gatewayClassName: my-gateway-class
  listeners:
    - name: https
      protocol: HTTPS
      port: 443
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: my-pfx-cert
            namespace: default
```

## 어노테이션

| 어노테이션 | 필수 | 설명 | 기본값 |
|------------|------|------|--------|
| `pfx-tls.kubernetes.io/convert` | 예 | 변환 활성화 (`"true"`이어야 함) | - |
| `pfx-tls.kubernetes.io/pfx-key` | 아니오 | PFX 데이터를 포함하는 Secret의 키 | `"pfx"` |
| `pfx-tls.kubernetes.io/password` | 아니오 | PFX 파일의 비밀번호 | `""` (빈 문자열) |
| `pfx-tls.kubernetes.io/password-secret-name` | 아니오 | 비밀번호를 포함하는 Secret 이름 | - |
| `pfx-tls.kubernetes.io/password-secret-key` | 아니오 | 비밀번호 Secret의 키 | - |
| `pfx-tls.kubernetes.io/converted` | 아니오 | 마커 어노테이션 (operator가 추가) | - |

## 아키텍처

Operator는 세 가지 주요 구성 요소로 이루어져 있습니다:

1. **Controller** (`pkg/controller/controller.go`): Secret을 감시하고 변환을 조정
2. **Converter** (`pkg/converter/converter.go`): PFX에서 PEM으로의 변환 로직 처리
3. **Main** (`main.go`): 진입점 및 Kubernetes 클라이언트 설정

### 작동 방식

1. Operator가 클러스터의 모든 Secret을 감시
2. `pfx-tls.kubernetes.io/convert: "true"` 어노테이션이 있는 Secret이 감지되면:
   - 지정된 키에서 PFX 데이터 추출
   - 비밀번호 검색 (지정된 경우)
   - PFX를 PEM 형식으로 변환
   - PEM 데이터로 Secret 업데이트
   - Secret을 변환됨으로 표시

## 개발

### 사전 요구 사항

- Go 1.21 이상
- Docker (컨테이너 이미지 빌드용)
- Kubernetes 클러스터로 구성된 kubectl

### 빌드 및 테스트

```bash
# 코드 포맷
make fmt

# 린터 실행
make vet

# 테스트 실행
make test

# 바이너리 빌드
make build

# 빌드 아티팩트 정리
make clean
```

### 로컬 개발

Kubernetes 클러스터에 대해 로컬로 Operator 실행:

```bash
go run main.go -kubeconfig=$HOME/.kube/config
```

## 보안 고려 사항

- **비밀번호 저장**: 프로덕션 환경에서는 어노테이션에 비밀번호를 저장하지 마세요. 대신 Secret 참조를 사용하세요.
- **RBAC**: Operator는 모든 네임스페이스에서 Secret을 읽고 업데이트할 수 있는 권한이 필요합니다.
- **Secret 타입**: 변환 후에도 원본 PFX 데이터는 PEM 데이터와 함께 Secret에 남아 있습니다.

## Azure Key Vault 통합

Azure Key Vault에서 PFX 인증서를 사용하는 경우, [Azure Key Vault Provider for Secrets Store CSI Driver](https://github.com/Azure/secrets-store-csi-driver-provider-azure)를 사용하여 PFX 인증서를 Secret으로 동기화한 다음 이 Operator로 변환할 수 있습니다.

## 기여

기여를 환영합니다! 이슈나 Pull Request를 자유롭게 제출해 주세요.

## 라이선스

이 프로젝트는 MIT 라이선스에 따라 라이선스가 부여됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.
