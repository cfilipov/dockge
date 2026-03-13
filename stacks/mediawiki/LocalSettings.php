<?php
# MediaWiki LocalSettings.php stub for Docker Compose
# Complete the installation wizard to generate a full configuration

$wgSitename = "My Wiki";
$wgServer = "http://localhost:8080";
$wgScriptPath = "";

$wgDBtype = "mysql";
$wgDBserver = "db";
$wgDBname = "mediawiki";
$wgDBuser = "mediawiki";
$wgDBpassword = "mediawiki_secret";

$wgSecretKey = "change_this_to_a_random_secret_key";
$wgUpgradeKey = "change_this_upgrade_key";

$wgLanguageCode = "en";
$wgLocaltimezone = "UTC";
