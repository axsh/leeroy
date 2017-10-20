Name:		leeroy
Version:	0.0.1
Release:	1%{?dist}
Summary:	axsh leeroy

Group:		Applications/Engineering
License:	None
URL:		https://github.com/axsh/leeroy/
Source:		https://github.com/axsh/leeroy
#Source0:	
BuildArch:	%{_build_arch}

%description
Jenkins integration with GitHub pull requests

%build
cd "${GOPATH}/src/github.com/axsh/leeroy"
(
  go build .
)

%install
cd "${GOPATH}/src/github.com/axsh/leeroy"
mkdir -p -p "$RPM_BUILD_ROOT"/opt/axsh/leeroy/bin
cp leeroy "$RPM_BUILD_ROOT"/opt/axsh/leeroy/bin

%files
%dir /opt/axsh/leeroy
%dir /opt/axsh/leeroy/bin
/opt/axsh/leeroy/bin/leeroy

%changelog

