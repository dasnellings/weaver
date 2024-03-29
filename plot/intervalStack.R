#!/usr/bin/env Rscript
depPackages = c("ggplot2", "hrbrthemes", "plotly", "htmlwidgets", "tidyr", "ggdendro", "reshape2", "grid", "heatmaply")
## Check load available dependencies, else install & load
package.check <- lapply(
  depPackages,
  FUN = function(x) {
    if (!require(x, character.only = TRUE)) {
      install.packages(x, dependencies = TRUE)
      library(x, character.only = TRUE)
    }
  }
)

data <- read.csv("roh.csv")

roh_chr7 <- data[ which(data$Chromosome == "chr7"), ]
roh_chr3 <- data[ which(data$Chromosome == "chr3"), ]

p_chr7 <- ggplot(roh_chr3, aes(x=Pos, color=StartOrEnd)) + 
  geom_histogram(aes(y=..density..), binwidth = 10, fill = "white", position="dodge") +
  geom_density(adjust = 1/5)
p_chr7

