#!/bin/bash -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
NOCOLOR='\033[0m'
RED='\033[0;31m'
GREEN='\033[0;32m'
ORANGE='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
LIGHTGRAY='\033[0;37m'
DARKGRAY='\033[1;30m'
LIGHTRED='\033[1;31m'
LIGHTGREEN='\033[1;32m'
YELLOW='\033[1;33m'
LIGHTBLUE='\033[1;34m'
LIGHTPURPLE='\033[1;35m'
LIGHTCYAN='\033[1;36m'
WHITE='\033[1;37m'

VENOM="${VENOM:-`which venom`}"
VENOM_OPTS="${VENOM_OPTS:---log debug --output-dir ./results --strict --stop-on-failure}"

CDS_API_URL="${CDS_API_URL:-http://localhost:8081}"
CDS_UI_URL="${CDS_UI_URL:-http://localhost:4200}"
CDSCTL="${CDSCTL:-`which cdsctl`}"
CDSCTL_CONFIG="${CDSCTL_CONFIG:-.cdsrc}"
SMTP_MOCK_URL="${SMTP_MOCK_URL-http://localhost:2024}"

echo -e "Using venom using following variables:"
echo -e "  VENOM=${CYAN}${VENOM}${NOCOLOR}"
echo -e "  VENOM_OPTS=${CYAN}${VENOM_OPTS}${NOCOLOR}"
echo ""

echo -e "Running tests using following variables:"
echo -e "  CDS_API_URL=${CYAN}${CDS_API_URL}${NOCOLOR}"
echo -e "  CDS_UI_URL=${CYAN}${CDS_UI_URL}${NOCOLOR}"
echo -e "  CDSCTL=${CYAN}${CDSCTL}${NOCOLOR}"
echo -e "  CDSCTL_CONFIG=${CYAN}${CDSCTL_CONFIG}${NOCOLOR}"
echo ""

smoke_tests() {
    echo "Running smoke tests:"
    for f in $(ls -1 00_*.yml); do
        echo -e "  ${YELLOW}${f}${NOCOLOR}"
        ${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL}  --var smtpmock.url=${SMTP_MOCK_URL} >${f}.output 2>&1
    done
}

initialization_tests() {
    echo "Running initialization tests:"
    echo -e "  ${YELLOW}01_signup.yml (admin)${NOCOLOR}"
    ${VENOM} run ${VENOM_OPTS} 01_signup.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var username=cds.integration.tests.rw --var email=it-user-rw@localhost.local --var fullname="IT User RW" --var smtpmock.url=${SMTP_MOCK_URL} >01_signup_admin.yml.output 2>&1
    echo -e "  ${YELLOW}01_signup.yml (user)${NOCOLOR}"
    ${VENOM} run ${VENOM_OPTS} 01_signup.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_user --var api.url=${CDS_API_URL} --var username=cds.integration.tests.ro --var email=it-user-ro@localhost.local --var fullname="IT User RO" --var smtpmock.url=${SMTP_MOCK_URL} >01_signup_user.yml.output 2>&1
}

cli_tests() {
     echo "Running CLI tests:"
    for f in $(ls -1 02_cli*.yml); do
        echo -e "  ${YELLOW}${f}${NOCOLOR}"
        ${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL}  --var smtpmock.url=${SMTP_MOCK_URL} >${f}.output 2>&1
    done
}

rm -rf ./results
mkdir results

smoke_tests
initialization_tests
cli_tests


