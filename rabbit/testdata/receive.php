#!/usr/bin/env php
<?php
$message = file_get_contents( 'php://stdin' ) ;
// Decode to get original value
$original = json_decode($message);


exit(1);