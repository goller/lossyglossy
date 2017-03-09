package main

import (
	"reflect"
	"testing"
)

func TestLatestItem(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		want     string
		wantErr  bool
	}{
		{
			name: "Test S3 RSS Channel",
			response: []byte(`
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Amazon Simple Storage Service (US Standard) Service Status</title>
    <link>http://status.aws.amazon.com/</link>
    <link rel="alternate" href="http://status.aws.amazon.com/rss/all.rss" type="application/rss+xml" title="Amazon Web Services Status Feed"/>
    <title type="text">Current service status feed for Amazon Simple Storage Service (US Standard).</title>
    <language>en-us</language>
    <pubDate>Wed,  8 Mar 2017 13:16:09 PST</pubDate>
    <updated>Wed,  8 Mar 2017 13:16:09 PST</updated>
    <generator>AWS Service Health Dashboard RSS Generator</generator>
    <ttl>5</ttl>

	 
	 <item>
	  <title type="text">Service disruption: [RESOLVED] Increased Error Rates</title>
	  <link>http://status.aws.amazon.com/</link>
	  <pubDate>Tue, 28 Feb 2017 14:11:00 PST</pubDate>
	  <guid>http://status.aws.amazon.com/#s3-us-standard_1488319860</guid>
	  <description>As of 1:49 PM PST, we are fully recovered for operations for adding new objects in S3, which was our last operation showing a high error rate. The Amazon S3 service is operating normally.</description>
	 </item>
	<item>
	<title type="text">Service disruption: [RESOLVED] Increased Error Rates</title>
	<link>http://status.aws.amazon.com/</link>
	<pubDate>Tue, 28 Feb 2017 13:13:00 PST</pubDate>
	<guid>http://status.aws.amazon.com/#s3-us-standard_1488316380</guid>
	<description>S3 object retrieval, listing and deletion are fully recovered now. We are still working to recover normal operations for adding new objects to S3.</description>
	</item>
  </channel>
</rss>
`),
			want: "Service disruption: [RESOLVED] Increased Error Rates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LatestItem(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("LatestItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LatestItem() = %v, want %v", got, tt.want)
			}
		})
	}
}
