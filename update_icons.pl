#!/usr/bin/perl -w
use strict;
use File::Slurp qw/read_file write_file/;
use Data::Dumper;

if( ! -e "repos" ) {
  mkdir "repos"
}

if( ! -e "repos/iconify-json" ) {
  `git clone https://github.com/iconify/collections-json repos/iconify-json`
} else {
  #chdir "repos/iconify-json";
  #`git pull`;
}

if( ! -e "repos/ujsonin" ) {
  `git clone https://github.com/nanoscopic/ujsonin repos/ujsonin`;
} else {
  #chdir "repos/jsonin";
  #`git pull`;
}
eval {
  use lib "./repos/ujsonin/perl/mod";
  use Ujsonin;
};
Ujsonin::init();

my @icons = qw/
  mdi-home
  mdi-anvil
  mdi-alarm-multiple
  mdi-view-dashboard-outline
  mdi-circle-box
  mdi-arrow-left-bold
  mdi-bomb
  mdi-format-list-checkbox
  mdi-cellphone-play
  mdi-clipboard-text
  mdi-cog
  mdi-account-supervisor
  mdi-account-circle
/;

my @sets = qw/mdi/;
my %sethash;
for my $set ( @sets ) {
  $sethash{ $set } = read_set( $set );
}

my $out;
$out .= "var iconify_icons = {\n";
for my $fullname ( @icons ) {
  if( $fullname =~ m/([a-z]+)-(.+)/ ) {
    my $set_name = $1;
    my $icon_name = "$set_name-$2";
    my $set = $sethash{ $set_name };
    my $info = $set->{ $icon_name };
    if( !$info ) {
      print "Missing $icon_name\n";
      next;
    }
    my $body = $info->{data};
    my $w = $info->{w};
    my $h = $info->{h};
    $body =~ s/`/"/g;
    $out .= "  \"$icon_name\": [$w,$h,'$body'],\n";
  }
}
$out .= "  \"end\":0\n";
$out .= "};\n";

write_file( "assets/js/iconify_icons.js", $out );

sub read_set {
  my $name = shift;
  my %hash;
  my $json_path = "repos/iconify-json/json/$name.json";
  my $json = read_file( $json_path );
  $json =~ s/\\"/`/g;
  my $root = Ujsonin::parse( $json );
  my $w = $root->{width};
  my $h = $root->{height};
  #print Dumper( $root );
  my $icons = $root->{ icons };
  for my $icon ( keys %$icons ) {
    #print "$icon\n";
    next if( $icon =~ /~\_/ );
    my $data = $icons->{ $icon };
    next if( ref( $data ) ne 'HASH' );
    $hash{ "$name-$icon" } = { data => $data->{body}, w => $w, h => $h };;
  }
  return \%hash;
}
