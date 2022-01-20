# ************************************************************
# Sequel Pro SQL dump
# Version 5438
#
# https://www.sequelpro.com/
# https://github.com/sequelpro/sequelpro
#
# Host: 127.0.0.1 (MySQL 5.5.5-10.5.8-MariaDB)
# Database: nocalhost
# Generation Time: 2020-11-26 06:57:56 +0000
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
  `context` text DEFAULT NULL,
  `user_id` int(11) NOT NULL DEFAULT 0,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `public` tinyint(1) DEFAULT 1,
  `status` tinyint(1) DEFAULT 1 COMMENT '1 enable, 0 disable',
  PRIMARY KEY (`id`),
  KEY `user_Id` (`user_id`),
  KEY `status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;



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



# Dump of table applications_users
# ------------------------------------------------------------

DROP TABLE IF EXISTS `applications_users`;

CREATE TABLE `applications_users` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `application_id` int(11) DEFAULT NULL,
  `user_id` int(11) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `application_id` (`application_id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;





# Dump of table clusters
# ------------------------------------------------------------

DROP TABLE IF EXISTS `clusters`;

CREATE TABLE `clusters` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) NOT NULL DEFAULT '',
  `marks` varchar(100) NOT NULL DEFAULT '',
  `user_id` int(11) NOT NULL DEFAULT 0,
  `server` varchar(500) NOT NULL DEFAULT '',
  `kubeconfig` text NOT NULL,
  `storage_class` varchar(100) NOT NULL DEFAULT '' COMMENT 'specify the k8s storage class',
  `info` text DEFAULT NULL COMMENT 'cluster extra info, such as versions, nodes',
  `deleted_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;



# Dump of table clusters_users
# ------------------------------------------------------------

DROP TABLE IF EXISTS `clusters_users`;

CREATE TABLE `clusters_users` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `application_id` int(11) NOT NULL,
  `cluster_id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL,
  `space_name` varchar(100) DEFAULT NULL COMMENT 'dev space name',
  `kubeconfig` text DEFAULT NULL COMMENT 'service account',
  `memory` int(11) DEFAULT NULL COMMENT 'memory limit',
  `cpu` int(11) DEFAULT NULL COMMENT 'CPU limit',
  `namespace` varchar(30) DEFAULT NULL,
  `status` tinyint(4) NOT NULL DEFAULT 0 COMMENT '0 not deployed, 1 deployed',
  `created_at` datetime DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `cluster_id` (`cluster_id`),
  KEY `user_id` (`user_id`),
  KEY `application_id` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;



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
	(1,'nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:latest',NULL),
	(2,'nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-wait:latest',NULL),
	(3,'nocalhost-docker.pkg.coding.net/nocalhost/bookinfo/productpage:latest',NULL),
	(4,'nocalhost-docker.pkg.coding.net/nocalhost/bookinfo/reviews:latest',NULL),
	(5,'nocalhost-docker.pkg.coding.net/nocalhost/bookinfo/details:latest',NULL),
	(6,'nocalhost-docker.pkg.coding.net/nocalhost/bookinfo/ratings:latest',NULL);

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
  `avatar` varchar(255) NOT NULL DEFAULT '',
  `phone` bigint(20) NOT NULL DEFAULT 0 ,
  `email` varchar(100) NOT NULL DEFAULT '',
  `is_admin` tinyint(4) NOT NULL DEFAULT 0,
  `status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '1 enable, 0 disable',
  `deleted_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;

INSERT INTO `users` (`id`, `uuid`, `username`, `name`, `password`, `avatar`, `phone`, `email`, `is_admin`, `status`, `deleted_at`, `created_at`, `updated_at`)
VALUES
	(1,'36882544-3bf5-4065-86a7-9b2188d71a1b','Admin','Admin','$2a$10$XkuHQPH9jJ6GZ3GL9IR8U.7xN0gH6zSiO5fIQIfESZ8eagPo/Jnii','',0,'admin@admin.com',1,1,NULL,'2020-10-13 16:22:20','2020-10-13 16:22:20');

/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;



/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
