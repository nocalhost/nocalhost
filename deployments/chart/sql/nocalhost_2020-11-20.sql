# ************************************************************
# Sequel Pro SQL dump
# Version 5438
#
# https://www.sequelpro.com/
# https://github.com/sequelpro/sequelpro
#
# Host: 127.0.0.1 (MySQL 5.7.22-log)
# Database: nocalhost
# Generation Time: 2020-11-20 06:24:39 +0000
# ************************************************************


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
SET NAMES utf8mb4;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;


# Dump of table applications
# ------------------------------------------------------------

DROP TABLE IF EXISTS `applications`;

CREATE TABLE `applications` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `context` text,
  `user_id` int(11) NOT NULL DEFAULT '0',
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `status` tinyint(1) DEFAULT '1' COMMENT '1启用，0禁用',
  PRIMARY KEY (`id`),
  KEY `user_Id` (`user_id`),
  KEY `status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

LOCK TABLES `applications` WRITE;
/*!40000 ALTER TABLE `applications` DISABLE KEYS */;

INSERT INTO `applications` (`id`, `context`, `user_id`, `created_at`, `updated_at`, `deleted_at`, `status`)
VALUES
	(1,'123',4,'2020-10-26 13:41:19','2020-10-26 13:41:19','2020-10-26 13:41:19',1),
	(3,'{\"application_url\":\"git@github.com:nocalhost/bookinfo.git\",\"application_name\":\"应用名\",\"source\":\"git\",\"install_type\":\"manifest\",\"resource_dir\":\"manifest\"}',4,'2020-10-26 13:41:19','2020-10-28 11:18:25',NULL,1),
	(4,'{\"application_url\":\"git@github.com:nocalhost/bookinfo.git\",\"application_name\":\"应用名\",\"source\":\"git\",\"install_type\":\"manifest\",\"resource_dir\":\"manifest\"}',4,'2020-10-26 13:41:19','2020-10-27 20:17:15',NULL,1);

/*!40000 ALTER TABLE `applications` ENABLE KEYS */;
UNLOCK TABLES;


# Dump of table applications_clusters
# ------------------------------------------------------------

DROP TABLE IF EXISTS `applications_clusters`;

CREATE TABLE `applications_clusters` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `application_id` int(11) DEFAULT NULL,
  `cluster_id` int(11) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `application_id` (`application_id`),
  KEY `cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

LOCK TABLES `applications_clusters` WRITE;
/*!40000 ALTER TABLE `applications_clusters` DISABLE KEYS */;

INSERT INTO `applications_clusters` (`id`, `application_id`, `cluster_id`, `created_at`, `deleted_at`, `updated_at`)
VALUES
	(4,3,1,'2020-10-28 15:06:31',NULL,'2020-10-28 15:06:31'),
	(5,4,3,'2020-10-28 15:06:31',NULL,'2020-10-28 15:06:31'),
	(6,4,8,'2020-10-28 15:06:31',NULL,'2020-10-28 15:06:31');

/*!40000 ALTER TABLE `applications_clusters` ENABLE KEYS */;
UNLOCK TABLES;


# Dump of table clusters
# ------------------------------------------------------------

DROP TABLE IF EXISTS `clusters`;

CREATE TABLE `clusters` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) NOT NULL DEFAULT '',
  `marks` varchar(100) NOT NULL DEFAULT '',
  `user_id` int(11) NOT NULL DEFAULT '0',
  `server` varchar(500) NOT NULL DEFAULT '',
  `kubeconfig` text NOT NULL,
  `info` text COMMENT '集群额外信息JSON、Kubernetes 版本、Node 节点之类',
  `deleted_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

LOCK TABLES `clusters` WRITE;
/*!40000 ALTER TABLE `clusters` DISABLE KEYS */;

INSERT INTO `clusters` (`id`, `name`, `marks`, `user_id`, `server`, `kubeconfig`, `info`, `deleted_at`, `created_at`, `updated_at`)
VALUES
	(1,'123','12',4,'1','1',NULL,NULL,'2020-10-22 19:39:43','2020-10-22 19:39:43'),
	(2,'name','marks',3,'1','kube',NULL,NULL,'2020-10-22 19:42:20','2020-10-22 19:42:20'),
	(20,'255','255',4,'https://9.134.115.255:6443','apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1ekNDQWMrZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJd01Ea3dNakV4TkRFd05sb1hEVE13TURnek1URXhOREV3Tmxvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBS3NjClBzU2hwRTB4THM0MTJlWUxkSTl0MlpoZFI2dHdxMWQydCt0MmF6VGg2SUIxVnBnT3BHc0hCdTgvemxxL0dENkkKcGpPYlZMV2k3eDM4U2dGbGc2VWdtZUFkV0IwZFhZV3UvTTdMRnRZUEpNcU5rUi92YUF0UlBnbFZlc3F0VHVQRApuQktuTCt5czFJZmh4WFFIVUxnWmpta0dUSlFvTjl5YXZzUS9zc0paTGlyL1V6UlFWR0U4VTVKRzVjNERPUEZVCjB5QjVvQndiRDhmSWVzSEZaVDVpY1Y3cFhscUZCR0pwU2JvUzdRMEpGZVBVdGNRcFNyRWxIUVhTUEVUb2Z1YngKSEtVeGVlRDFmRDg1a3Riemh6K2xzVWdGb29VM0oxZTZOV3NDZjRaSStNZ2x3eFh1UmRGN2N0YUd4THIyVThZMAo0dFl4dTF4QjUyOGIvLzB1UE5zQ0F3RUFBYU5DTUVBd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZNUUNFdG95QU12V1B6UjVQYnM5bThqWnNsSlBNQTBHQ1NxR1NJYjMKRFFFQkN3VUFBNElCQVFDVmtZQ2c4a0t4MkFaNXJjWDhlY0pyZTNMNm5MUlZUZndKN25WNGY2T0F1bkhpNVZnawpWdG9WVDdkOUVrZ1A3RlBMWHE1Z085K1FTN1FSNGZ3TnVOTUZKVm5XSWw2YnJsa3pnb25TVFdLUGRCVlJQZm1sCi9nZE9ZWW0xS3VxYk9JZ2tMMkhPVE90UE9EdENkWGtISW1ndDc3VkJNTDMzZ3FZSk16QkNVR0p0U05wanpyNUoKSjdsV0p5QlM4TG90UzZ1NnI4RUxHdDBBd3FRdFNzZlZhSCt6OW9JMXl1VDdhSlBuSG56ZUpySFlPNGg1Q3BaRApKY2ljSVhvR0JmS3pRU2YrcmZJNXdsQWxjMkZsS2NHbFl1d2k5L2owaGNzUzIzWHQwclpXaXRqdXFoRVEwcmhNClFtRWgvbW1NRjJWR3pYWHZBMEh6V3BmNldwWmJ4SzVlU2R4dQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==\n    server: https://9.134.115.255:6443\n  name: kubernetes\ncontexts:\n- context:\n    cluster: kubernetes\n    user: kubernetes-admin\n  name: kubernetes-admin@kubernetes\ncurrent-context: kubernetes-admin@kubernetes\nkind: Config\npreferences: {}\nusers:\n- name: kubernetes-admin\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURFekNDQWZ1Z0F3SUJBZ0lJS3EwMUpERGZ3dXd3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TURBNU1ESXhNVFF4TURaYUZ3MHlNVEE1TURJeE1UUXhNRGxhTURReApGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1Sa3dGd1lEVlFRREV4QnJkV0psY201bGRHVnpMV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQTRKTmxFNDlFVjE5TVRmL3cKbm9VNnVaMW50blpMYzZGV0h1L0J1WWtmaFB0TUJkRXRscHd1SUZqZzRrWFppOFhLLzNoc0doemptVktsSThGMwpmZDhwUVlOU1p0MUhiVjVQZVhaektGR3d0NmJCTDZ0b1U1NVBjNXRBVWw5WHRHWHdueWJTbUF2cGN0VE9FTCtxCkVvSUhmWEpqalcwZDlZNGo4cytkWEFhbW1QWWlYWi9yOWNwRjBiL0dILzJtZjhpdDB2RmJuclo0UXdQUTQ3TmEKWlJPYi9IWmtoMmR0eFpsMTZTbU9Nc1hkWG9hQXZwTjk2MXhuVXAzZ3RLQk4rYlkxaHBScVNYRVJZVmRVOW1JLwozRzV4a1htNEhBQkF0TnZmazI0UjRvUGhjc3IreStXaWVDeC9qL3NiMUh6UlNvZnlvSjlteTdHZVlkR1drS1o4ClRHOVptUUlEQVFBQm8wZ3dSakFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUgKQXdJd0h3WURWUjBqQkJnd0ZvQVV4QUlTMmpJQXk5WS9OSGs5dXoyYnlObXlVazh3RFFZSktvWklodmNOQVFFTApCUUFEZ2dFQkFDSHZXMkxLemVydGNydlVqQTZWd1Q5ZVBXVUd6bzZJcXZXKzg5R3RocHpJeVVxMDdya0h0eEhQCm9lMThHOGprazNDTXdUWm1oNTFDVnBBaEN2SFIvRWpYWmZkT2Ewc1R6Yk1TSlgra2M3RjMrVUF2QnRzd0YybU4KSCtoZ1hmZlFGaUxwcFFBZjUwSlQ2NDZWc1AwRE9DNGM4dHR1dWhIQVR3YUZJTzQvc2k0bFN1alRxbDNIVzlSVApOZ0o4cjhwbGlPbjdQaCsyL1ZYUDlqeXFnVzFhY2JIRVBqYkttamJQQlJvR2Rua3pHS0pJWWY1dkpGa1ZRVVJBCmZEVVJUeXo3UmpkQzIwamhrYnV3NG10czQzRE41K0NUN1RlOUZxMVdabWxNNWpHNkVxSkphbENNZFRoaHdkRGsKbWJGZnhoVVllU1Z1VTJEUFQ1OHJIWDFrbTN6c1RrRT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=\n    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBNEpObEU0OUVWMTlNVGYvd25vVTZ1WjFudG5aTGM2RldIdS9CdVlrZmhQdE1CZEV0Cmxwd3VJRmpnNGtYWmk4WEsvM2hzR2h6am1WS2xJOEYzZmQ4cFFZTlNadDFIYlY1UGVYWnpLRkd3dDZiQkw2dG8KVTU1UGM1dEFVbDlYdEdYd255YlNtQXZwY3RUT0VMK3FFb0lIZlhKampXMGQ5WTRqOHMrZFhBYW1tUFlpWFovcgo5Y3BGMGIvR0gvMm1mOGl0MHZGYm5yWjRRd1BRNDdOYVpST2IvSFpraDJkdHhabDE2U21PTXNYZFhvYUF2cE45CjYxeG5VcDNndEtCTitiWTFocFJxU1hFUllWZFU5bUkvM0c1eGtYbTRIQUJBdE52ZmsyNFI0b1BoY3NyK3krV2kKZUN4L2ovc2IxSHpSU29meW9KOW15N0dlWWRHV2tLWjhURzlabVFJREFRQUJBb0lCQUNNZXlkclNOK1RXRVcvTgpTOHJ1bU8xNE1VVDJvUHdYU2dtU2d5QkowblVRZTZZWlBXRGxVYzFiT09nSjltaUdhU1drcG5zNjgxa0I5TE52CnlRa1ZRalZ0blJCYklKVjQvMExHaEdIVXpLY2IyL0JoaFBJMnVzUWdqbUdUYVhyYnlsS0pWcnZTZVJLdE53Q2wKaUtwV1RXZVA0UU80QWN4cUN6TW94cm9pakNFMWFiTk1hMnF1N2dkeUR5d1laY04rMWtoN09tL3lqc2drL2I1LwpnNHhjdXJSbjVlc0UwR1pUUGJSVnZtL1dmSENDWXhxVFEvVWRuZFY2bWZHbjUrV2tGN2pPMlBleGo0OWxoWmhxCmQ0QmViaFRSeEJ2YTQrRUoraktwWU1LWEF5WkdxUE5qbU1weFk4RUtrNyt3VzBMZjNNSU5hR3RiMDJIaWtYV0kKbHdBUENvRUNnWUVBNHFqbFZsKzJPRE84bTdwSEVRL0J1dnUxRVdkRUtZV2pkTzY4RHAyUXdBRlRVajF3VmxVMgpYZmp2WU02MTJuQ2dZMlZQajBjSGs2dmk2REVhTVBmREJxR0RiOUJDWDFTU2JKNktqdDN4Qkw5NmgxZ0NtV0Q4ClRRK1B3TjlaRGEwczg1cFFXTG1EdkNuYUxsZ1ord0k5anBYaG1pLzZOb1JvNDhCZnRGQ2Y0Y2tDZ1lFQS9hVncKZE9NdFNjZXNiU2puQlFyTkZIcHQ1Zm10Q2FoczdRcktWWmxMZXpaeDNueU5iSWttT0tVdWw5dVZuTEc3S1gwcwpzSFpJVUdPblJ3blBRcHhqczBHRlZPNHFsNy9DZ29pL2ZWNWJmbCttc2UvQW5TblFXdEsxc1EzRGYrTnlKdEYwCk5tK01tNnhQQkhXRHRacjIvNDZ5UDJOaWd3WHpGOTlKZlk3T0lWRUNnWUEwcGhTM2JuNE9LZjVha2ZkbUFDbjQKKy9UQU9TTjlIWnl0VWJML0ZoeUViUXBrcFA3T0h2Y0U5d2pyakxoektBd3BhbVFEblBVbW1SdVk0YWI2enVKUApUUDhSM3VjNzY1SWpodVFhY1hWRnJCQ1RGWjlzN3psTDBSeU1LWlV1OXhYazgraEw0N08wNW1mV3NnSSs2dk5QCmhvTWo1SmNUU01od2RzUUVSMklMK1FLQmdRQ0FNZlJ6YnpvOWR1aWp4eTl6c2ZEU3I4b0ptTFluRW5QekhpZ0QKT1ZZWDhQMStLRTlHRXM4NWcrclhuNGl2U0hqQzBGd2MxN3RXdmZjV2hWTzJZOXBVQ0FKK1dWMDNreGlZNXNwNQpiNDRvZ2VsN055U1Bpa21mRGEzOHpXc0lvUWpacTdUanFsOVRjclFCR2UrMmdwcmhzTnBRQlVnTjFwejFiTW4wCjVvOHg4UUtCZ0J4WFdTZUQ4Z2J1TVFmak5QOGRUVTJiVVkvb0o1cWZLOGVRODU0ZTBhSGg3ZGtPbVVPaGppV1IKcTNIY1ppSDFGbDlVV1BRNVlpZE94cUdwb3FDeUtadkpUWmt3d1V6czR4OU9yR05tVFBVZWJUelJDeFYrVDY5MQpoZlREdjdVb2lvTXoxMC9ZREtHSFVHYWZ4VkNZUkRrK0FZRzRkWmFEWWUyWFFFcm04V0VtCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==','{\"cluster_version\":\"v1.19.0\",\"nodes\":\"2\"}',NULL,'2020-11-17 15:31:21','2020-11-17 15:31:21');

/*!40000 ALTER TABLE `clusters` ENABLE KEYS */;
UNLOCK TABLES;


# Dump of table clusters_users
# ------------------------------------------------------------

DROP TABLE IF EXISTS `clusters_users`;

CREATE TABLE `clusters_users` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `application_id` int(11) NOT NULL COMMENT '应用 ID',
  `cluster_id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL,
  `kubeconfig` text COMMENT 'serviceAccount',
  `memory` int(11) DEFAULT NULL COMMENT '内存限制',
  `cpu` int(11) DEFAULT NULL COMMENT 'CPU 限制',
  `namespace` varchar(30) DEFAULT NULL COMMENT '随机生成的命名空间',
  `status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0未部署，1已部署',
  `created_at` datetime DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `cluster_id` (`cluster_id`),
  KEY `user_id` (`user_id`),
  KEY `application_id` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

LOCK TABLES `clusters_users` WRITE;
/*!40000 ALTER TABLE `clusters_users` DISABLE KEYS */;

INSERT INTO `clusters_users` (`id`, `application_id`, `cluster_id`, `user_id`, `kubeconfig`, `memory`, `cpu`, `namespace`, `status`, `created_at`, `deleted_at`, `updated_at`)
VALUES
	(33,3,20,4,'apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1ekNDQWMrZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJd01Ea3dNakV4TkRFd05sb1hEVE13TURnek1URXhOREV3Tmxvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBS3NjClBzU2hwRTB4THM0MTJlWUxkSTl0MlpoZFI2dHdxMWQydCt0MmF6VGg2SUIxVnBnT3BHc0hCdTgvemxxL0dENkkKcGpPYlZMV2k3eDM4U2dGbGc2VWdtZUFkV0IwZFhZV3UvTTdMRnRZUEpNcU5rUi92YUF0UlBnbFZlc3F0VHVQRApuQktuTCt5czFJZmh4WFFIVUxnWmpta0dUSlFvTjl5YXZzUS9zc0paTGlyL1V6UlFWR0U4VTVKRzVjNERPUEZVCjB5QjVvQndiRDhmSWVzSEZaVDVpY1Y3cFhscUZCR0pwU2JvUzdRMEpGZVBVdGNRcFNyRWxIUVhTUEVUb2Z1YngKSEtVeGVlRDFmRDg1a3Riemh6K2xzVWdGb29VM0oxZTZOV3NDZjRaSStNZ2x3eFh1UmRGN2N0YUd4THIyVThZMAo0dFl4dTF4QjUyOGIvLzB1UE5zQ0F3RUFBYU5DTUVBd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZNUUNFdG95QU12V1B6UjVQYnM5bThqWnNsSlBNQTBHQ1NxR1NJYjMKRFFFQkN3VUFBNElCQVFDVmtZQ2c4a0t4MkFaNXJjWDhlY0pyZTNMNm5MUlZUZndKN25WNGY2T0F1bkhpNVZnawpWdG9WVDdkOUVrZ1A3RlBMWHE1Z085K1FTN1FSNGZ3TnVOTUZKVm5XSWw2YnJsa3pnb25TVFdLUGRCVlJQZm1sCi9nZE9ZWW0xS3VxYk9JZ2tMMkhPVE90UE9EdENkWGtISW1ndDc3VkJNTDMzZ3FZSk16QkNVR0p0U05wanpyNUoKSjdsV0p5QlM4TG90UzZ1NnI4RUxHdDBBd3FRdFNzZlZhSCt6OW9JMXl1VDdhSlBuSG56ZUpySFlPNGg1Q3BaRApKY2ljSVhvR0JmS3pRU2YrcmZJNXdsQWxjMkZsS2NHbFl1d2k5L2owaGNzUzIzWHQwclpXaXRqdXFoRVEwcmhNClFtRWgvbW1NRjJWR3pYWHZBMEh6V3BmNldwWmJ4SzVlU2R4dQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==\n    server: https://9.134.115.255:6443\n  name: nocal4odxb\ncontexts:\n- context:\n    cluster: nocal4odxb\n    namespace: nocal4odxb\n    user: nocalhost-dev-account\n  name: nocal4odxb\ncurrent-context: nocal4odxb\nkind: Config\npreferences: {}\nusers:\n- name: nocalhost-dev-account\n  user:\n    token: eyJhbGciOiJSUzI1NiIsImtpZCI6Iko2YlE4RWliZVZMN3NaTVhCY19jNk9CWHFlR3FybldIRjFGaF9WeHd4WWMifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJub2NhbDRvZHhiIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZWNyZXQubmFtZSI6Im5vY2FsaG9zdC1kZXYtYWNjb3VudC10b2tlbi12emRtaiIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJub2NhbGhvc3QtZGV2LWFjY291bnQiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC51aWQiOiJhOWMzZmRiYy03M2E5LTRhNTMtOTM2Zi01ZTYxODI0MjU4OGEiLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6bm9jYWw0b2R4Yjpub2NhbGhvc3QtZGV2LWFjY291bnQifQ.Tl5ga2xwUBaaDYF_Ggt7a9gaplt7M0I9p_yB84_-0dxasSXSlzDYfbyNrwZw54nX_DRE13GqiXJDjTj3EdosrGHo2zjaE9yuqD3IyFCzWZCjL3GD1wUvQ4iRLfbfR92hdzF51lXB2IvLIkEsymVkF69t9CDpCsnd_Ig3VQQ8pxh0OOKggrs4YilsnsnBKCy9XLF51ysqP2qNOQdXAV5LYOyyup2hcC-i0lHhCPXyMyW4C9i9DqOxmmCU86gdqntu0ZwezHcPt48i5dCW7UpaQr0FfZmUtZNbjf3mWLtVdEIGnaG2JObJTDCDeRH7aOB3gjhhCjAa7giKjlJNtz8Mpw\n',0,0,'nocal4odxb',1,'2020-11-18 14:51:54',NULL,'2020-11-19 21:25:00');

/*!40000 ALTER TABLE `clusters_users` ENABLE KEYS */;
UNLOCK TABLES;


# Dump of table pre_pull
# ------------------------------------------------------------

DROP TABLE IF EXISTS `pre_pull`;

CREATE TABLE `pre_pull` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `images` varchar(1000) NOT NULL DEFAULT '',
  `deleted_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

LOCK TABLES `pre_pull` WRITE;
/*!40000 ALTER TABLE `pre_pull` DISABLE KEYS */;

INSERT INTO `pre_pull` (`id`, `images`, `deleted_at`)
VALUES
	(1,'codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:latest',NULL),
	(2,'codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-wait:latest',NULL),
	(3,'codingcorp-docker.pkg.coding.net/nocalhost/bookinfo/productpage:latest',NULL),
	(4,'codingcorp-docker.pkg.coding.net/nocalhost/bookinfo/reviews:latest',NULL),
	(5,'codingcorp-docker.pkg.coding.net/nocalhost/bookinfo/details:latest',NULL),
	(6,'codingcorp-docker.pkg.coding.net/nocalhost/bookinfo/ratings:latest',NULL);

/*!40000 ALTER TABLE `pre_pull` ENABLE KEYS */;
UNLOCK TABLES;


# Dump of table users
# ------------------------------------------------------------

DROP TABLE IF EXISTS `users`;

CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uuid` varchar(100) NOT NULL DEFAULT '',
  `username` varchar(255) NOT NULL DEFAULT '',
  `name` varchar(20) DEFAULT NULL,
  `password` varchar(60) NOT NULL DEFAULT '',
  `avatar` varchar(255) NOT NULL DEFAULT '' COMMENT '头像',
  `phone` bigint(20) NOT NULL DEFAULT '0' COMMENT '手机号',
  `email` varchar(100) NOT NULL DEFAULT '' COMMENT '邮箱',
  `is_admin` tinyint(4) NOT NULL DEFAULT '0' COMMENT '内置管理员',
  `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '状态，1正常，0禁用',
  `deleted_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='用户表';

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;

INSERT INTO `users` (`id`, `uuid`, `username`, `name`, `password`, `avatar`, `phone`, `email`, `is_admin`, `status`, `deleted_at`, `created_at`, `updated_at`)
VALUES
	(1,'5c6f2386-b841-4c72-91db-43cd787dcc41','admin','1','$2a$10$WhJY.MCtsp5kmnyl/UAdQuWbbMzxvmLCPeDhcpxyL84lYey829/ym','',1,'a@qq.com',0,1,NULL,'2020-05-05 11:11:11','2020-05-05 11:11:11'),
	(2,'5c6f2386-b841-4c72-91db-43cd787dcc42','','1','$2a$10$SfAVmftfCrcEP/CgTL3j0uxkzMUr.KEo5DQWmxJk5uvgrSqmp2tuS','',0,'434533508@qq.com',0,1,NULL,'2020-10-13 13:24:42','2020-10-13 13:24:42'),
	(4,'36882544-3bf5-4065-86a7-9b2188d71a1b','','管理员','$2a$10$XkuHQPH9jJ6GZ3GL9IR8U.7xN0gH6zSiO5fIQIfESZ8eagPo/Jnii','',0,'4345335081@qq.com',1,1,NULL,'2020-10-13 16:22:20','2020-10-13 16:22:20'),
	(5,'3d6e0510-8a83-42c9-8ac1-a6e28469776f','','王炜2','$2a$10$sHmZNH5eGEqahOkadDwzKePf3sRhOl2tKku/rBj4IlTkxxFLMtFm2','',0,'41@qq.com',0,1,NULL,'2020-11-03 11:38:56','2020-11-17 10:26:28'),
	(6,'823605c6-e29a-4590-9fc9-3da5e37488d4','','王炜','$2a$10$cDvGNDpg36ih/FIXvUoko.fTaKOgYflacCIIs.OsRais7W5oD1z4K','',0,'412@qq.com',0,1,NULL,'2020-11-03 12:31:16','2020-11-03 12:31:16'),
	(8,'0c97d745-4de0-4c05-9f3d-e45d802da901','','王炜','$2a$10$CCD0X2GQNBIadxn1d4DjGOJpmQ4/1.qSErKsYv080EQlzsBn1ZHoa','',0,'string1',0,1,'2020-11-03 14:44:38','2020-11-03 12:32:10','2020-11-03 14:20:53');

/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;



/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
