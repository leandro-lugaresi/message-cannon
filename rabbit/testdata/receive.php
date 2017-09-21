#!/usr/bin/env php
<?php
$message = file_get_contents( 'php://stdin' ) ;
// Decode to get original value
$test = json_decode($message);


exit($test['exitCode']);