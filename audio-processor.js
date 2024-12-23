// audio-processor.js
class TTSAudioProcessor extends AudioWorkletProcessor {
    constructor() {
      super();
      this.audioQueue = [];
      this.isPlaying = false;
  
      // Set up message handling
      this.port.onmessage = (e) => {
        if (e.data.type === 'audio') {
          // Add new audio data to the queue
          this.audioQueue.push(new Float32Array(e.data.audioData));
        }
      };
    }
  
    process(inputs, outputs, parameters) {
      const output = outputs[0];
      const outputChannel = output[0];
  
      if (this.audioQueue.length === 0) {
        return true;
      }
  
      // Get the next chunk of audio data
      let samplesNeeded = outputChannel.length;
      let outputIndex = 0;
  
      while (samplesNeeded > 0 && this.audioQueue.length > 0) {
        const currentBuffer = this.audioQueue[0];
        const remaining = currentBuffer.length - (currentBuffer.position || 0);
        const copyCount = Math.min(samplesNeeded, remaining);
  
        // Copy samples to output
        for (let i = 0; i < copyCount; i++) {
          outputChannel[outputIndex++] = currentBuffer[currentBuffer.position + i];
        }
  
        // Update positions
        currentBuffer.position = (currentBuffer.position || 0) + copyCount;
        samplesNeeded -= copyCount;
  
        // Remove finished buffers
        if (currentBuffer.position >= currentBuffer.length) {
          this.audioQueue.shift();
        }
      }
  
      // Fill any remaining output with silence
      while (outputIndex < outputChannel.length) {
        outputChannel[outputIndex++] = 0;
      }
  
      return true;
    }
  }
  
  // Register the processor
  registerProcessor('tts-audio-processor', TTSAudioProcessor);
  