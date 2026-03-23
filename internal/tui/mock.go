package tui

import "Karazhan/internal/server"

func mockServerData() []server.ESMNode {
	return []server.ESMNode{
		{
			ESMName: "Data Lake Storage", ESMCode: "NE032765",
			ESMFullName: "NHN Cloud>Cloud_상품>Data & Analytics>Data Lake Storage",
			CIServerList: []server.CIServer{
				// cdpwkr-hdp: CommonDataPlatform + DataLakeStorage-알파
				mockSrv("cdpwkr-hdp011", "10.161.79.21", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp012", "10.161.79.22", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp013", "10.161.79.23", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp014", "10.161.79.24", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp015", "10.161.79.25", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp016", "10.161.79.26", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp017", "10.161.79.27", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp018", "10.161.79.28", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),
				mockSrv("cdpwkr-hdp019", "10.161.79.29", "NCC1", "Ubuntu 22.04",
					sg("CommonDataPlatform", "서비스", "APP", "", nil),
					sg("DataLakeStorage-알파", "개발", "APP", "", nil)),

				// dlsa-dlsapp: DataLakeStorage-리얼-광주 #s3-proxy
				mockSrv("dlsa-dlsapp-g1001", "10.140.103.114 ; 10.140.113.184", "광주AI센터", "Ubuntu 22.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"s3-proxy"})),
				mockSrv("dlsa-dlsapp-g1002", "10.140.103.115 ; 10.140.113.185", "광주AI센터", "Ubuntu 22.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"s3-proxy"})),
				mockSrv("dlsa-dlsapp-g1003", "10.140.103.116 ; 10.140.113.186", "광주AI센터", "Ubuntu 22.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"s3-proxy"})),

				// dlsa-dlsapps: DataLakeStorage-베타-광주 #s3-proxy
				mockSrv("dlsa-dlsapps-g1001", "10.140.103.117 ; 10.140.113.187", "광주AI센터", "Ubuntu 22.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"s3-proxy"})),

				// dlsa-haproxy: DataLakeStorage-리얼-광주 #HAProxy
				mockSrv("dlsa-haproxy-g1001", "10.140.113.181 ; 10.17.165.41 ; 114.110.136.71", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"HAProxy"})),
				mockSrv("dlsa-haproxy-g1002", "10.140.113.182 ; 10.17.165.42 ; 114.110.136.72", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"HAProxy"})),

				// dlsa-haproxys: DataLakeStorage-베타-광주
				mockSrv("dlsa-haproxys-g1001", "10.140.113.183 ; 10.17.165.43", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", nil)),

				// dlsa-kr3alpha: DataLakeStorage-알파-광주
				mockSrv("dlsa-kr3alpha-g1901", "10.140.69.38", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-알파-광주", "개발", "APP", "WMI, KR3", []string{"HAProxy"})),
				mockSrv("dlsa-kr3alpha-g1902", "10.140.69.46", "광주AI센터", "Ubuntu 22.04",
					sg("DataLakeStorage-알파-광주", "개발", "APP", "WMI, KR3", []string{"Nginx", "console-api", "discovery", "internal-gateway", "s3-proxy", "tempo"})),
				mockSrv("dlsa-kr3alpha-g1903", "10.140.69.47", "광주AI센터", "Ubuntu 22.04",
					sg("DataLakeStorage-알파-광주", "개발", "APP", "WMI, KR3", []string{"OTel Collector", "admin-backend", "discovery", "metering-billing", "metering-consumer", "s3-proxy", "spring admin"})),
				mockSrv("dlsa-kr3alpha-g1904", "10.140.69.42", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-알파-광주", "개발", "APP", "WMI, KR3", []string{"clickhouse keeper", "clickhouse server"})),

				// dlsa-kr3beta: DataLakeStorage-베타-광주
				mockSrv("dlsa-kr3beta-g1901", "10.140.69.40", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"OTel Collector", "console-api", "discovery", "metering-billing"})),
				mockSrv("dlsa-kr3beta-g1902", "10.140.69.41", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"OTel Collector", "admin-backend", "metering-consumer", "spring admin", "tempo"})),
				mockSrv("dlsa-kr3beta-g1903", "10.140.69.51", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"NGINX", "console-api", "internal-gateway", "tempo"})),
				mockSrv("dlsa-kr3beta-g1904", "10.140.69.52", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"NGINX", "admin-backend", "discovery", "internal-gateway", "metering-consumer"})),
				mockSrv("dlsa-kr3beta-g1905", "10.140.69.43", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"clickhouse keeper", "clickhouse server"})),
				mockSrv("dlsa-kr3beta-g1906", "10.140.69.44", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"clickhouse keeper", "clickhouse server"})),
				mockSrv("dlsa-kr3beta-g1907", "10.140.69.45", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "APP", "", []string{"clickhouse keeper", "clickhouse server"})),

				// dlsa-kr3real: DataLakeStorage-리얼-광주
				mockSrv("dlsa-kr3real-g1901", "10.140.69.48", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"clickhouse keeper", "clickhouse server"})),
				mockSrv("dlsa-kr3real-g1902", "10.140.69.49", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"clickhouse keeper", "clickhouse server"})),
				mockSrv("dlsa-kr3real-g1903", "10.140.69.50", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"clickhouse keeper", "clickhouse server"})),
				mockSrv("dlsa-kr3real-g1904", "10.140.69.53", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"OTel Collector", "console-api", "discovery", "internal-gateway"})),
				mockSrv("dlsa-kr3real-g1905", "10.140.69.54", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"admin-backend", "internal-gateway", "metering-consumer", "spring admin", "tempo"})),
				mockSrv("dlsa-kr3real-g1906", "10.140.69.55", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"NGINX", "OTel Collector", "console-api", "console-web", "metering-billing"})),
				mockSrv("dlsa-kr3real-g1907", "10.140.69.56", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "APP", "KR3 리전 물리 서버 구성", []string{"NGINX", "admin-backend", "batch", "console-web", "discovery", "metering-consumer", "tempo"})),

				// dlsd-kr3dev: 베타+알파 DB #valkey
				mockSrv("dlsd-kr3dev-g1901", "10.140.143.71", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"valkey"}),
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"valkey"})),
				mockSrv("dlsd-kr3dev-g1902", "10.140.143.72", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"valkey"}),
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"valkey"})),
				mockSrv("dlsd-kr3dev-g1903", "10.140.143.73", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"valkey"}),
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"valkey"})),

				// dlsd-kr3real: 리얼 DB #valkey
				mockSrv("dlsd-kr3real-g1901", "10.140.143.104", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"valkey"})),
				mockSrv("dlsd-kr3real-g1902", "10.140.143.105", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"valkey"})),
				mockSrv("dlsd-kr3real-g1903", "10.140.143.106", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"valkey"})),

				// dlsg-kr3alpha: 알파 DB #mongo
				mockSrv("dlsg-kr3alpha-g1901", "10.140.133.38", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"mongo"})),
				mockSrv("dlsg-kr3alpha-g1902", "10.140.133.39", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"mongo"})),
				mockSrv("dlsg-kr3alpha-g1903", "10.140.133.40", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"mongo"})),

				// dlsg-kr3beta: 베타 DB #mongo
				mockSrv("dlsg-kr3beta-g1901", "10.140.143.77", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"mongo"})),
				mockSrv("dlsg-kr3beta-g1902", "10.140.143.78", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"mongo"})),
				mockSrv("dlsg-kr3beta-g1903", "10.140.143.79", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"mongo"})),

				// dlsg-kr3real: 리얼 DB #mongo
				mockSrv("dlsg-kr3real-g1001", "10.140.165.21", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"mongo"})),
				mockSrv("dlsg-kr3real-g1002", "10.140.165.22", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"mongo"})),
				mockSrv("dlsg-kr3real-g1003", "10.140.165.23", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"mongo"})),

				// dlsk-kr3dev: 베타+알파 DB #kafka
				mockSrv("dlsk-kr3dev-g1901", "10.140.143.74", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"kafka"}),
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"kafka"})),
				mockSrv("dlsk-kr3dev-g1902", "10.140.143.75", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"kafka"}),
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"kafka"})),
				mockSrv("dlsk-kr3dev-g1903", "10.140.143.76", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-베타-광주", "개발", "DB", "", []string{"kafka"}),
					sg("DataLakeStorage-알파-광주", "개발", "DB", "", []string{"kafka"})),

				// dlsk-kr3real: 리얼 DB #kafka
				mockSrv("dlsk-kr3real-g1901", "10.140.143.80", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"kafka"})),
				mockSrv("dlsk-kr3real-g1902", "10.140.143.81", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"kafka"})),
				mockSrv("dlsk-kr3real-g1903", "10.140.143.82", "광주AI센터", "Ubuntu 24.04",
					sg("DataLakeStorage-리얼-광주", "서비스", "DB", "", []string{"kafka"})),
			},
		},
	}
}

func sg(name, usage, useType, remark string, tags []string) server.ServerGroup {
	return server.ServerGroup{
		ServerGroupName:   name,
		ServerUsageName:   usage,
		ServerUseTypeName: useType,
		ServerGroupRemark: remark,
		ServerTags:        tags,
	}
}

func mockSrv(host, ip, idc, os string, groups ...server.ServerGroup) server.CIServer {
	return server.CIServer{
		HostName:        host,
		IP:              ip,
		IDCName:         idc,
		OSName:          os,
		ServerState:     "가동중",
		ManagerName:     "유시형",
		ServerGroupList: groups,
		ESMList: []server.ESMRef{{
			ESMName:     "Data Lake Storage",
			ESMFullName: "NHN Cloud>Cloud_상품>Data & Analytics>Data Lake Storage",
			ESMCode:     "NE032765",
		}},
	}
}
