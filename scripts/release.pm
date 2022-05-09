package release;

use Exporter;

our @EXPORT = qw(
    version_ok
    slurp_top_entry
    capture_run
    slurp_run
    run
    progress_item
    progress_bullet
    progress_status
    progress_quit

    changelog_file
    dist_dir
    s3base_url
);

use constant changelog_file => 'CHANGELOG.md';
use constant dist_dir => 'dist';
use constant s3base_url => 's3://garotate.qubling.cloud/';

sub version_ok {
    my ($version) = @_;
    return $version && $version =~ /^
        v\d+                      # major
        \.
        \d+                       # minor
        (?:-(?:alpha|beta|rc)\d+) # keyword
    $/x;
}

sub git_has_tag {
    my ($tag) = @_;

    my $tout = slurp_run('git', 'tag', '-l', $tag);
    $tout =~ s/^\s+//; $vout =~ s/\s+$//;

    return $tout eq $tag;
}

sub verify_tag {
    my ($version) = @_;

    die "Tag named $version is already in use.\n"
        if git_has_tag($version);
}

sub slurp_top_entry {
    my ($file) = @_;

    open my $fh, '<', $file
        or progress_quit("cannot read $file: $!");

    my $notes = '';
    my $remainder = '';
    my $s = 'start';
    while (<$fh>) {
        $s eq 'start' && do {
            next if /^# /;
            next if /^\s*$/;
            $s = 'notes';
            next;
        };
        $s eq 'notes' && do {
            if (/^## /) {
                $s = 'remainder';
                $remainder .= $_;
                next;
            }
            $notes .= $_;
            next;
        };

        $remainder .= $_;
    }

    close $file;

    return ($notes, $remainder);
}

sub capture_run {
    my ($cmd, $callback) = @_;

    open my $rh, '-|', @$cmd
        or progress_quit("cannot run '@$cmd': $!");

    while (<$rh>) {
        $callback->($_);
    }

    close $rh
        or progress_quit("cannot complete '@$cmd': $!");
}

sub slurp_run {
    my $content = '';
    capture_run(
        \@_,
        sub { $content .= $_ },
    );
    return $content;
}

sub run {
    capture_run(\@_, sub {});
}

sub progress_item {
    print @_;
}

sub progress_bullet {
    say " - ", @_;
}

sub progress_status {
    say @_;
}

sub progress_quit {
    say "FAILED";
    confess @_;
}

