# Cognito Authentication with Naver

AWS Cognito와 네이버 로그인 연동 하는법을 설명한다. 이를 위해서는 아래 설명을 따르도록한다.

## 네이버 어플리케이션 생성
네이버 개발자 센터에 방문하여 로그인 연동에 필요한 어플리케이션을 생성하고 CLIENT_ID와 CLIENT_SECRET를 받아오도록한다.
1. 네이버 개발자 센터 방문: https://developers.naver.com/main/
2. 어플리케이션 관리 이동: https://developers.naver.com/apps/#/list
3. 아래 값을 입력 할 때, http://localhost:8080 를 반드시 지키도록 한다.
![](/img/naver-dev-app1.png)
![](/img/naver-dev-app2.png)

## Cognito 자원 생성
Cognito 자원 생성에는 Terraform(https://www.terraform.io/) 을 사용한다.

아래와 같이 `terraform` 명령어를 통해 Cognito를 생성하고, Outputs에 나온 값을 메모하도록한다.

```shell
$ cd terraform

$ terraform init
$ terraform apply -auto-approve
..
aws_cognito_user_pool_client.this: Modifying... [id=xxxxyyyyyyyyzzzzzzzz]
aws_cognito_user_pool_client.this: Modifications complete after 0s [id=xxxxyyyyyyyyzzzzzzzz]

Apply complete! Resources: 0 added, 1 changed, 0 destroyed.

Outputs:

user_pool_client_id = "xxxxyyyyyyyyzzzzzzzz"
user_pool_id = "ap-northeast-2_hxxxxxxx"
```
## 서버 코드 빌드 및 실행
Docker container를 이용해 빌드하고 바이너리를 실행 할 수 있도록 구성했다.
```shell
$ docker build -t cognito-with-naver .
[+] Building 34.6s (10/10) FINISHED
 => [internal] load build definition from Dockerfile                                                                                                         0.0s
 => => transferring dockerfile: 198B                                                                                                                         0.0s
 => [internal] load .dockerignore                                                                                                                            0.0s
 => => transferring context: 2B                                                                                                                              0.0s
 => [internal] load metadata for docker.io/library/golang:1.17                                                                                               1.0s
 => [internal] load build context                                                                                                                            0.0s
 => => transferring context: 7.65kB                                                                                                                          0.0s
 => [1/5] FROM docker.io/library/golang:1.17@sha256:6556ce40115451e40d6afbc12658567906c9250b0fda250302dffbee9d529987                                         0.0s
 => CACHED [2/5] WORKDIR /go/src/naver                                                                                                                       0.0s
 => [3/5] COPY . .                                                                                                                                           0.7s
 => [4/5] RUN go get -d -v ./...                                                                                                                            21.1s
 => [5/5] RUN go install -v ./...                                                                                                                            9.2s
 => exporting to image                                                                                                                                       2.4s
 => => exporting layers                                                                                                                                      2.4s
 => => writing image sha256:66420db31ded665799cc50c03030d0ec6fd187c98cc7e8ee07cd53a06054abe9                                                                 0.0s
 => => naming to docker.io/library/cognito-with-naver
 ```
 
빌드된 이미지와 네이버 어플리케이션 정보, Cognito 정보를 이용해 컨테이너를 실행한다. 설정을 위해 4가지 환경 변수가 이용된다.
* NAVER_CLIENT_ID
* NAVER_CLIENT_SECRET
* COGNITO_USER_POOL_ID
* COGNITO_USER_POOL_CLIENT_ID
```shell
$ docker run --rm -p 8080:8080 -e NAVER_CLIENT_ID=XXX -e NAVER_CLIENT_SECRET=XXX -e COGNITO_USER_POOL_ID=XXX -e COGNITO_USER_POOL_CLIENT_ID=XXX cognito-with-naver
```
 
## 웹 페이지 접속
웹 브라우저에서 http://localhost:8080 를 접속한다. `Login with Naver` 버튼을 클릭해 로그인이 제대로 동작하는지 확인한다.
브라우저에 입력하는 주소가 네이버 개발자 센터에 입력한 http://localhost:8080 과 일치하는지 확인하도록 한다.
![](/img/localhost1.png)

## 네이버 정보 가져오기
제대로 로그인이 되었다면 상단에 Naver Access Token에 네이버 인증 결과로부터 받은 Access Token이 입력되어있다.
이제 이 Access Token을 이용해 네이버로부터 이메일, 이름 정보를 가져오도록한다.
![](/img/localhost2.png)

## Cognito와 네이버 연동
연동 흐름을 좀 더 쉽게 이해하기 위해 버튼으로 각각 로직을 나누어서 구현했다. 실제로 서비스에서 사용하기 위해서는 여기 정리된 내용을 적절하게 조합해서 용도에 맞게 사용해야한다.

### Cognito User Pool에 사용자 존재여부 체크
네이버에서 가져온 이메일 주소를 기반으로, Cognito User Pool에 사용자가 있는지 확인한다.
`Check user exists in Cognito` 버튼을 클릭하면, 아래 함수가 실행되게된다.
https://github.com/odg0318/cognito-with-naver/blob/main/main.go#L245

### Cognito에 가입 및 로그인
사용자가 없으면 가입을 먼저 진행한다. 가입시 네이버에서 받아온 이메일과 이름을 이용해 자동으로 사용자를 생성하도록 한다.
그 후 가입된 Cognito 사용자로 로그인을하여 Cognito Access Token을 받아온다.
![](/img/localhost3.png)

### Cognito 동작 확인
위에서 받아온 Cognito Access Token 동작을 확인하기 위해, [Cognito > GetUser](https://docs.aws.amazon.com/cognito-user-identity-pools/latest/APIReference/API_GetUser.html) API를 호출해보도록한다.
![](/img/localhost4.png)
